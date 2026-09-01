package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/backup"
	"github.com/boweeb/hasp/internal/adapter/fswrite"
	"github.com/boweeb/hasp/internal/adapter/sshconfig"
	"github.com/boweeb/hasp/internal/domain"
)

func newHostApplier(dir string) *Applier {
	return &Applier{FS: fswrite.New(), Backups: backup.New(dir)}
}

// TestNewHostUseCase_Plan_DefaultGroup_NoPriorConfig covers the simplest case: no ~/.ssh/config
// exists yet, no --group given. Plan produces one WriteRegion Change with an empty Before and a
// well-formed brand-new region as After; applying it and re-parsing must round-trip cleanly.
func TestNewHostUseCase_Plan_DefaultGroup_NoPriorConfig(t *testing.T) {
	dir := t.TempDir()

	uc := NewHostUseCase{}
	plan, err := uc.Plan(NewHostRequest{
		KeyDir:   dir,
		Patterns: []string{"foobarco-prod"},
		HostName: "foobarco.example.com",
		User:     "deploy",
		Port:     "2222",
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	c, ok := plan.Changes[0].(WriteRegion)
	if !ok {
		t.Fatalf("Changes[0] = %T, want WriteRegion", plan.Changes[0])
	}
	if len(c.Before) != 0 {
		t.Errorf("Before = %q, want empty", c.Before)
	}
	if len(plan.Witnesses) != 0 {
		t.Errorf("Witnesses = %+v, want none (nothing existed to race against)", plan.Witnesses)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	configPath := filepath.Join(dir, "config")
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	parsed := sshconfig.Parse(written)
	if len(parsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %+v, want none", parsed.MarkerDefects)
	}
	region := soleMarkedRegion(t, parsed)
	hb := soleHostBlock(t, region.Body)
	if len(hb.Patterns) != 1 || hb.Patterns[0] != "foobarco-prod" {
		t.Errorf("Patterns = %v, want [foobarco-prod]", hb.Patterns)
	}
	assertDirectiveValue(t, hb, "HostName", "foobarco.example.com")
	assertDirectiveValue(t, hb, "User", "deploy")
	assertDirectiveValue(t, hb, "Port", "2222")

	if got := sshconfig.Parse(written).Nodes; !bytes.Equal(sshconfig.RenderNodes(got), written) {
		t.Errorf("Render(Parse(written)) != written; round-trip broken")
	}
}

// TestNewHostUseCase_Plan_DefaultGroup_PreservesHandWrittenContent exercises roadmap.md M3 exit
// criterion 1's spirit for this one verb: an existing ~/.ssh/config with hand-written content
// outside any marked region, plus a marked region already holding one HostBlock, gets a second
// HostBlock appended inside the region — and every byte outside hasp's markers is unchanged.
func TestNewHostUseCase_Plan_DefaultGroup_PreservesHandWrittenContent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	// The hand-written Host block deliberately sits *after* the region rather than before it:
	// ssh_config has no way back to top-level scope once a Host/Match line has appeared outside
	// any hasp-opened region (sshconfig's own scanMarkers doc comment), so a region placed after
	// an unclosed human Host block would itself be (correctly) flagged nested — this fixture
	// exercises hand-written content, not that unrelated defect.
	original := "# a human wrote this\n\n" +
		sshconfig.ManagedRegionBegin + "\n" +
		"Host existing\n    User carol\n" +
		sshconfig.ManagedRegionEnd + "\n" +
		"\nHost bastion\n    HostName bastion.example.com\n"
	writeFile(t, configPath, original)

	uc := NewHostUseCase{}
	plan, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"newhost"}, User: "dave"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Witnesses) != 1 {
		t.Fatalf("len(Witnesses) = %d, want 1 (a region already existed)", len(plan.Witnesses))
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	const humanPrefix = "# a human wrote this\n\n"
	const humanSuffix = "\nHost bastion\n    HostName bastion.example.com\n"
	if !bytes.HasPrefix(written, []byte(humanPrefix)) {
		t.Fatalf("content before the marked region changed; got prefix %q", written[:min(len(written), len(humanPrefix))])
	}
	if !bytes.HasSuffix(written, []byte(humanSuffix)) {
		t.Fatalf("content after the marked region changed; got suffix %q", written[max(0, len(written)-len(humanSuffix)):])
	}

	parsed := sshconfig.Parse(written)
	if len(parsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %+v, want none", parsed.MarkerDefects)
	}
	region := soleMarkedRegion(t, parsed)
	var patterns []string
	for _, n := range region.Body {
		if hb, ok := n.(*sshconfig.HostBlock); ok {
			patterns = append(patterns, hb.Patterns[0])
		}
	}
	if len(patterns) != 2 || patterns[0] != "existing" || patterns[1] != "newhost" {
		t.Errorf("region HostBlock patterns = %v, want [existing newhost]", patterns)
	}
}

// TestNewHostUseCase_Plan_CustomGroup_NewFile covers --group where the group file doesn't exist
// yet: Plan produces two Changes, group file first (T20 "least recoverable last"), then the
// Include line in ~/.ssh/config's own region.
func TestNewHostUseCase_Plan_CustomGroup_NewFile(t *testing.T) {
	dir := t.TempDir()

	uc := NewHostUseCase{}
	plan, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"foobarco-prod"}, Group: "work"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2", len(plan.Changes))
	}

	groupFilePath := filepath.Join(dir, "work.sshconfig")
	first, ok := plan.Changes[0].(WriteRegion)
	if !ok || first.File != groupFilePath {
		t.Fatalf("Changes[0] = %+v, want a WriteRegion targeting %s", plan.Changes[0], groupFilePath)
	}
	configPath := filepath.Join(dir, "config")
	second, ok := plan.Changes[1].(WriteRegion)
	if !ok || second.File != configPath {
		t.Fatalf("Changes[1] = %+v, want a WriteRegion targeting %s", plan.Changes[1], configPath)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	groupBytes, err := os.ReadFile(groupFilePath)
	if err != nil {
		t.Fatalf("read %s: %v", groupFilePath, err)
	}
	if !bytes.HasPrefix(groupBytes, []byte("# hasp:owned")) {
		t.Errorf("group file does not start with the hasp:owned header: %q", groupBytes[:min(len(groupBytes), 40)])
	}
	groupParsed := sshconfig.Parse(groupBytes)
	if len(groupParsed.MarkerDefects) != 0 {
		t.Fatalf("group file MarkerDefects = %+v, want none", groupParsed.MarkerDefects)
	}
	hb := findHostBlock(t, groupParsed.Nodes, "foobarco-prod")
	if hb == nil {
		t.Fatal("group file has no Host foobarco-prod stanza")
	}
	if !bytes.Equal(sshconfig.RenderNodes(groupParsed.Nodes), groupBytes) {
		t.Error("group file Render(Parse(written)) != written; round-trip broken")
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	configParsed := sshconfig.Parse(configBytes)
	region := soleMarkedRegion(t, configParsed)
	var includeCount int
	for _, n := range region.Body {
		if d, ok := n.(*sshconfig.Directive); ok && d.Keyword == "Include" {
			includeCount++
			args := d.Args()
			if len(args) != 1 || args[0] != groupFilePath {
				t.Errorf("Include args = %v, want [%s]", args, groupFilePath)
			}
		}
	}
	if includeCount != 1 {
		t.Errorf("Include directive count = %d, want exactly 1", includeCount)
	}
}

// TestNewHostUseCase_Plan_IncludeStaysDiscoverable_AfterExistingDefaultGroupHost is a regression
// test for a real bug caught by manual smoke testing: when ~/.ssh/config's region already holds a
// HostBlock (from an earlier default-group `new host` call) and a later `new host --group=...`
// call appends a new Include directive naively at the end of the region body, the Include line
// physically lands *after* the HostBlock with no intervening Host/Match header to close its scope
// — ssh_config's own grammar then groups that Include into the preceding HostBlock's own
// Directives on the next Parse (there is no way back to top-level scope), silently scoping it to
// only that Host pattern matching. scan.LoadConfigTree's includeTargets deliberately never looks
// inside a HostBlock for an Include to follow, so the whole custom group silently stops being
// discovered — a P1 "never create dead configuration" violation reachable only through this
// specific call ordering (custom group added *after* a default-group host already exists), which
// TestNewHostUseCase_Plan_CustomGroup_NewFile's own empty-starting-config fixture never exercised.
// This test drives the full stack (Plan, Apply, Derive) exactly the way a real user session would,
// rather than inspecting the region's raw node order directly, so it fails the same way the
// original bug actually manifested.
func TestNewHostUseCase_Plan_IncludeStaysDiscoverable_AfterExistingDefaultGroupHost(t *testing.T) {
	dir := t.TempDir()
	uc := NewHostUseCase{}
	applier := newHostApplier(dir)

	firstPlan, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"foobarco-prod"}, HostName: "foobarco.example.com"})
	if err != nil {
		t.Fatalf("first Plan: %v", err)
	}
	if _, err := applier.Apply(firstPlan); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	secondPlan, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"foobarco-staging"}, Group: "work", Port: "2222"})
	if err != nil {
		t.Fatalf("second Plan: %v", err)
	}
	if _, err := applier.Apply(secondPlan); err != nil {
		t.Fatalf("second Apply: %v", err)
	}

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Hosts) != 2 {
		t.Fatalf("got %d hosts, want 2 (foobarco-prod, foobarco-staging): %+v", len(m.Hosts), m.Hosts)
	}
	if findHostByPattern(m.Hosts, "foobarco-staging") == nil {
		t.Errorf("foobarco-staging was not discovered — the Include to its group file was likely swallowed into the preceding HostBlock's own scope")
	}
}

// TestNewHostUseCase_Plan_CustomGroup_ExistingFile covers --group where the group file (and its
// Include line) already exist from a prior `new host` call: Plan has exactly one Change, and no
// duplicate Include line is added.
func TestNewHostUseCase_Plan_CustomGroup_ExistingFile(t *testing.T) {
	dir := t.TempDir()

	uc := NewHostUseCase{}
	first, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"foobarco-prod"}, Group: "work"})
	if err != nil {
		t.Fatalf("first Plan: %v", err)
	}
	applier := newHostApplier(dir)
	if _, err := applier.Apply(first); err != nil {
		t.Fatalf("first Apply: %v", err)
	}

	second, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"foobarco-staging"}, Group: "work"})
	if err != nil {
		t.Fatalf("second Plan: %v", err)
	}
	if len(second.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1 (Include line already present)", len(second.Changes))
	}
	groupFilePath := filepath.Join(dir, "work.sshconfig")
	c, ok := second.Changes[0].(WriteRegion)
	if !ok || c.File != groupFilePath {
		t.Fatalf("Changes[0] = %+v, want a WriteRegion targeting %s", second.Changes[0], groupFilePath)
	}
	if len(second.Witnesses) != 1 {
		t.Fatalf("len(Witnesses) = %d, want 1", len(second.Witnesses))
	}

	if _, err := applier.Apply(second); err != nil {
		t.Fatalf("second Apply: %v", err)
	}

	groupBytes, err := os.ReadFile(groupFilePath)
	if err != nil {
		t.Fatalf("read %s: %v", groupFilePath, err)
	}
	groupParsed := sshconfig.Parse(groupBytes)
	if findHostBlock(t, groupParsed.Nodes, "foobarco-prod") == nil {
		t.Error("group file missing the first stanza (foobarco-prod)")
	}
	if findHostBlock(t, groupParsed.Nodes, "foobarco-staging") == nil {
		t.Error("group file missing the second stanza (foobarco-staging)")
	}

	configPath := filepath.Join(dir, "config")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	configParsed := sshconfig.Parse(configBytes)
	region := soleMarkedRegion(t, configParsed)
	var includeCount int
	for _, n := range region.Body {
		if d, ok := n.(*sshconfig.Directive); ok && d.Keyword == "Include" {
			includeCount++
		}
	}
	if includeCount != 1 {
		t.Errorf("Include directive count = %d, want exactly 1 (no duplicate)", includeCount)
	}
}

// TestNewHostUseCase_Plan_RejectsDuplicatePattern covers P1's guard: a requested pattern that
// already exists anywhere reachable from ~/.ssh/config must be refused, wrapping ErrUsage.
func TestNewHostUseCase_Plan_RejectsDuplicatePattern(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, "Host taken\n    HostName taken.example.com\n")

	uc := NewHostUseCase{}
	_, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"taken"}})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// TestNewHostUseCase_Plan_RejectsDuplicatePattern_InCustomGroup extends the duplicate guard to a
// pattern that lives inside a custom host group reached via Include.
func TestNewHostUseCase_Plan_RejectsDuplicatePattern_InCustomGroup(t *testing.T) {
	dir := t.TempDir()
	groupFilePath := filepath.Join(dir, "work.sshconfig")
	writeFile(t, groupFilePath, "Host taken\n    HostName taken.example.com\n")
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nInclude "+groupFilePath+"\n"+sshconfig.ManagedRegionEnd+"\n")

	uc := NewHostUseCase{}
	_, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"taken"}, Group: "other"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// TestNewHostUseCase_Plan_FailsClosedOnMarkerDefect covers T18: a malformed marker in
// ~/.ssh/config must refuse the whole Plan, not just proceed and hope for the best.
func TestNewHostUseCase_Plan_FailsClosedOnMarkerDefect(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	// An end marker with no matching begin: DefectEndBeforeBegin (region_test.go's own shape,
	// mirrored in check_host_test.go's TestCheck_MarkerDefect).
	writeFile(t, configPath, sshconfig.ManagedRegionEnd+"\n")

	uc := NewHostUseCase{}
	_, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"anything"}})
	if err == nil {
		t.Fatal("Plan succeeded against a malformed marker, want a refusal")
	}
}

// TestNewHostUseCase_Plan_RejectsEmptyRequest covers the basic usage-error guards.
func TestNewHostUseCase_Plan_RejectsEmptyRequest(t *testing.T) {
	uc := NewHostUseCase{}
	cases := []NewHostRequest{
		{Patterns: []string{"foo"}},                                  // no KeyDir
		{KeyDir: "/tmp"},                                             // no patterns
		{KeyDir: "/tmp", Patterns: []string{""}},                     // only empty patterns
		{KeyDir: "/tmp", Patterns: []string{"foo"}, Group: "config"}, // reserved group name
		{KeyDir: "/tmp", Patterns: []string{"foo"}, Group: "a/b"},    // group with separator
	}
	for i, req := range cases {
		if _, err := uc.Plan(req); !errors.Is(err, ErrUsage) {
			t.Errorf("case %d: err = %v, want ErrUsage", i, err)
		}
	}
}

// TestNewHostUseCase_Plan_RejectsControlCharactersInDirectiveValues is the regression test for
// review bug 1: an embedded '\n' (or '\r', or NUL) in HostName/User/Port/IdentityFile/Group/KeyDir
// must cause Plan to return an error wrapping ErrUsage *before* any Change is constructed —
// asserted by checking the returned Plan is the zero value, not merely that a bad Plan would later
// be caught. The first case reproduces the review's own concrete repro string: a HostName crafted
// to close hasp's own end marker early and inject an attacker-controlled Host stanza outside the
// tracked region, if it were ever allowed to reach buildNewHostBlock unvalidated. The last case is
// a later-round finding: KeyDir itself, once joined into groupFile (resolveHostGroupFile) and
// spliced into planIncludeChange's "Include <path>" directive, is the identical trust boundary —
// reproduced with the reviewer's own exact repro string, which also confirms the KeyDir defaults
// from os.UserHomeDir() (internal/cli/root.go), not merely operator typo, so environment tampering
// reaches it too.
func TestNewHostUseCase_Plan_RejectsControlCharactersInDirectiveValues(t *testing.T) {
	dir := t.TempDir()
	uc := NewHostUseCase{}
	cases := []struct {
		name string
		req  NewHostRequest
	}{
		{
			name: "hostname injects a fake end marker and a rogue Host block",
			req: NewHostRequest{KeyDir: dir, Patterns: []string{"newhost"},
				HostName: "evil\n# <<< hasp:managed <<<\nHost injected\n    User attacker"},
		},
		{name: "user contains a bare CR", req: NewHostRequest{KeyDir: dir, Patterns: []string{"newhost"}, User: "evil\r"}},
		{name: "port contains a NUL byte", req: NewHostRequest{KeyDir: dir, Patterns: []string{"newhost"}, Port: "22\x00"}},
		{name: "identity file contains a newline", req: NewHostRequest{KeyDir: dir, Patterns: []string{"newhost"}, IdentityFile: "~/.ssh/id\nHost injected"}},
		{name: "group contains a newline", req: NewHostRequest{KeyDir: dir, Patterns: []string{"newhost"}, Group: "work\nHost injected"}},
		{
			name: "key directory injects a fake end marker and a rogue Host block via the Include path",
			req: NewHostRequest{
				KeyDir:   dir + "\n# <<< hasp:managed <<<\nHost injected\n    User attacker",
				Group:    "work",
				Patterns: []string{"newhost"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := uc.Plan(tc.req)
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("Plan error = %v, want ErrUsage", err)
			}
			if plan.Summary != "" || plan.Changes != nil || plan.Witnesses != nil {
				t.Errorf("Plan = %+v, want the zero value — validation must run before any Change is built", plan)
			}
		})
	}
}

// TestNewHostUseCase_Plan_RejectsControlCharactersInPatterns is the regression test for this
// review round's bug 3: Patterns — the positional "Host <pattern...>" header arguments — was
// missed by bug 1's fix and is spliced unescaped into the "Host ..." header's RawValue exactly
// like HostName/User/Port/IdentityFile. Mirrors
// TestNewHostUseCase_Plan_RejectsControlCharactersInDirectiveValues's own shape: asserts Plan
// returns an error wrapping ErrUsage with the zero-value Plan, i.e. validation runs before any
// Change is constructed, not caught later on write. The first case reproduces the verifier's own
// concrete repro string: a pattern crafted to close hasp's own end marker early and inject an
// attacker-controlled Host stanza outside the tracked region.
func TestNewHostUseCase_Plan_RejectsControlCharactersInPatterns(t *testing.T) {
	dir := t.TempDir()
	uc := NewHostUseCase{}
	cases := []struct {
		name string
		req  NewHostRequest
	}{
		{
			name: "pattern injects a fake end marker and a rogue Host block",
			req: NewHostRequest{KeyDir: dir,
				Patterns: []string{"evil\n# <<< hasp:managed <<<\nHost injected\n    User attacker"}},
		},
		{name: "pattern contains a bare CR", req: NewHostRequest{KeyDir: dir, Patterns: []string{"evil\r"}}},
		{name: "pattern contains a NUL byte", req: NewHostRequest{KeyDir: dir, Patterns: []string{"evil\x00"}}},
		{name: "pattern contains an unquoted hash", req: NewHostRequest{KeyDir: dir, Patterns: []string{"evil#comment"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := uc.Plan(tc.req)
			if !errors.Is(err, ErrUsage) {
				t.Fatalf("Plan error = %v, want ErrUsage", err)
			}
			if plan.Summary != "" || plan.Changes != nil || plan.Witnesses != nil {
				t.Errorf("Plan = %+v, want the zero value — validation must run before any Change is built", plan)
			}
		})
	}
}

// TestNewHostUseCase_Plan_RejectsHashInDirectiveValues is the regression test for review finding
// 4: an unquoted '#' round-trips fine on the immediate Render, but a *subsequent* Parse of the
// written file reinterprets it as a trailing comment, silently truncating the value on next read.
// validateDirectiveValue rejects it outright (see its own doc comment for why rejecting was chosen
// over quoting) — this asserts that rejection wraps ErrUsage, before any Change is constructed.
func TestNewHostUseCase_Plan_RejectsHashInDirectiveValues(t *testing.T) {
	dir := t.TempDir()
	uc := NewHostUseCase{}
	plan, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"newhost"}, HostName: "evil#comment"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
	if plan.Changes != nil {
		t.Errorf("Plan = %+v, want the zero value", plan)
	}
}

// TestNewHostUseCase_Plan_DefaultGroup_FirstWriteWithPreExistingTopLevelHostBlock is the regression
// test for review bug 2: the very first `new host` write against a ~/.ssh/config that already has
// a pre-existing bare Host block (no hasp region yet at all) — the single most likely first-use
// shape on a real machine — must not self-corrupt hasp's own marker. Before the fix,
// planDefaultGroupChange appended the brand-new region at EOF, landing it textually after
// "Host bastion"; scanMarkers's inHostBlock latch (armed by that pre-region Host line) then
// misclassified hasp's own freshly-written markers as DefectNestedInHostBlock on the very next
// Parse, locking hasp out of the region it had just created (T18). This drives Plan, a real
// Applier, and a fresh re-Parse of the result — the actual failure mode, not just an inspection of
// the Plan's own bytes.
func TestNewHostUseCase_Plan_DefaultGroup_FirstWriteWithPreExistingTopLevelHostBlock(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, "Host bastion\n    HostName bastion.example.com\n")

	uc := NewHostUseCase{}
	plan, err := uc.Plan(NewHostRequest{KeyDir: dir, Patterns: []string{"newhost"}, User: "dave"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	parsed := sshconfig.Parse(written)
	if len(parsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %+v, want none (bug 2 regression)", parsed.MarkerDefects)
	}
	region := soleMarkedRegion(t, parsed)
	hb := soleHostBlock(t, region.Body)
	if len(hb.Patterns) != 1 || hb.Patterns[0] != "newhost" {
		t.Errorf("Patterns = %v, want [newhost]", hb.Patterns)
	}
	assertDirectiveValue(t, hb, "User", "dave")

	if !bytes.Contains(written, []byte("Host bastion\n    HostName bastion.example.com\n")) {
		t.Errorf("written = %q, want the pre-existing bastion block preserved verbatim", written)
	}
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
}

// --- test helpers ---

func soleMarkedRegion(t *testing.T, f *sshconfig.File) *sshconfig.MarkedRegion {
	t.Helper()
	var found *sshconfig.MarkedRegion
	for _, n := range f.Nodes {
		if r, ok := n.(*sshconfig.MarkedRegion); ok {
			if found != nil {
				t.Fatal("more than one MarkedRegion found, want exactly one")
			}
			found = r
		}
	}
	if found == nil {
		t.Fatal("no MarkedRegion found, want exactly one")
	}
	return found
}

func soleHostBlock(t *testing.T, nodes []sshconfig.Node) *sshconfig.HostBlock {
	t.Helper()
	var found *sshconfig.HostBlock
	for _, n := range nodes {
		if hb, ok := n.(*sshconfig.HostBlock); ok {
			if found != nil {
				t.Fatal("more than one HostBlock found, want exactly one")
			}
			found = hb
		}
	}
	if found == nil {
		t.Fatal("no HostBlock found, want exactly one")
	}
	return found
}

func findHostBlock(t *testing.T, nodes []sshconfig.Node, pattern string) *sshconfig.HostBlock {
	t.Helper()
	for _, n := range nodes {
		if hb, ok := n.(*sshconfig.HostBlock); ok {
			for _, p := range hb.Patterns {
				if p == pattern {
					return hb
				}
			}
		}
	}
	return nil
}

func findHostByPattern(hosts []domain.Host, pattern string) *domain.Host {
	for i, h := range hosts {
		for _, p := range h.Patterns {
			if p == pattern {
				return &hosts[i]
			}
		}
	}
	return nil
}

func assertDirectiveValue(t *testing.T, hb *sshconfig.HostBlock, keyword, want string) {
	t.Helper()
	for _, n := range hb.Directives {
		if d, ok := n.(*sshconfig.Directive); ok && d.Keyword == keyword {
			args := d.Args()
			if len(args) != 1 || args[0] != want {
				t.Errorf("%s args = %v, want [%s]", keyword, args, want)
			}
			return
		}
	}
	t.Errorf("no %s directive found, want value %q", keyword, want)
}
