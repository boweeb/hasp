package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/sshconfig"
)

// strp is a small test-only helper for building the *string values EditHostRequest's directive
// fields require to distinguish "untouched" (nil) from "set" (non-nil, non-empty) from "remove"
// (non-nil, empty).
func strp(s string) *string { return &s }

// TestEditHostUseCase_Plan_InPlace_DirectiveChange covers required test 1: a default-group stanza
// with a sibling stanza both before and after it, --hostname only. Only the target directive
// changes; everything else — including both siblings, byte-for-byte — is untouched, and the
// stanza stays at its original position in the region.
func TestEditHostUseCase_Plan_InPlace_DirectiveChange(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := "# top\n\n" +
		sshconfig.ManagedRegionBegin + "\n" +
		"Host before\n    User alice\n" +
		"Host target\n    HostName old.example.com\n    User bob\n" +
		"Host after\n    User carol\n" +
		sshconfig.ManagedRegionEnd + "\n" +
		"\n# bottom\n"
	writeFile(t, configPath, original)

	uc := EditHostUseCase{}
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "target", HostName: strp("new.example.com")})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	if len(plan.Witnesses) != 1 || plan.Witnesses[0].Path != configPath {
		t.Fatalf("Witnesses = %+v, want exactly one on %s", plan.Witnesses, configPath)
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
		t.Fatalf("MarkerDefects = %+v, want none", parsed.MarkerDefects)
	}

	if !bytes.HasPrefix(written, []byte("# top\n\n")) {
		t.Errorf("leading hand-written content not preserved: %q", written[:min(len(written), 20)])
	}
	if !bytes.HasSuffix(written, []byte("\n# bottom\n")) {
		t.Errorf("trailing hand-written content not preserved: %q", written[max(0, len(written)-20):])
	}
	if !bytes.Contains(written, []byte("Host before\n    User alice\n")) {
		t.Error("sibling stanza (before) not preserved byte-identically")
	}
	if !bytes.Contains(written, []byte("Host after\n    User carol\n")) {
		t.Error("sibling stanza (after) not preserved byte-identically")
	}

	region := soleMarkedRegion(t, parsed)
	if len(region.Body) != 3 {
		t.Fatalf("region.Body has %d nodes, want 3", len(region.Body))
	}
	var order []string
	for _, n := range region.Body {
		hb, ok := n.(*sshconfig.HostBlock)
		if !ok || len(hb.Patterns) != 1 {
			t.Fatalf("unexpected node in region.Body: %+v", n)
		}
		order = append(order, hb.Patterns[0])
	}
	if order[0] != "before" || order[1] != "target" || order[2] != "after" {
		t.Errorf("region.Body order = %v, want [before target after] (position must not change)", order)
	}

	target := findHostBlock(t, region.Body, "target")
	if got, ok := directiveValue(target, "HostName"); !ok || got != "new.example.com" {
		t.Errorf("target HostName = %q (ok=%v), want new.example.com", got, ok)
	}
	if got, ok := directiveValue(target, "User"); !ok || got != "bob" {
		t.Errorf("target User = %q (ok=%v), want bob (untouched)", got, ok)
	}
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
}

// TestEditHostUseCase_Plan_InPlace_DirectiveRemoval covers required test 2: HostName: &"" on a
// stanza that has one — it's gone, everything else round-trips clean.
func TestEditHostUseCase_Plan_InPlace_DirectiveRemoval(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := sshconfig.ManagedRegionBegin + "\nHost target\n    HostName old.example.com\n    User bob\n" + sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, original)

	uc := EditHostUseCase{}
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "target", HostName: strp("")})
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
		t.Fatalf("MarkerDefects = %+v, want none", parsed.MarkerDefects)
	}
	region := soleMarkedRegion(t, parsed)
	target := soleHostBlock(t, region.Body)
	if _, ok := directiveValue(target, "HostName"); ok {
		t.Error("HostName directive still present, want it removed")
	}
	if got, ok := directiveValue(target, "User"); !ok || got != "bob" {
		t.Errorf("User = %q (ok=%v), want bob (untouched)", got, ok)
	}
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
}

// TestEditHostUseCase_Plan_InPlace_DirectiveRemoval_Absent covers required test 3: Port: &"" on a
// stanza with no Port — a no-op, no error, file byte-identical afterward.
func TestEditHostUseCase_Plan_InPlace_DirectiveRemoval_Absent(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := sshconfig.ManagedRegionBegin + "\nHost target\n    User bob\n" + sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, original)

	uc := EditHostUseCase{}
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "target", Port: strp("")})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	c, ok := plan.Changes[0].(WriteRegion)
	if !ok {
		t.Fatalf("Changes[0] = %+v, want a WriteRegion", plan.Changes[0])
	}
	if !bytes.Equal(c.Before, c.After) {
		t.Errorf("Before != After for a no-op removal:\nBefore: %q\nAfter:  %q", c.Before, c.After)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	if !bytes.Equal(written, []byte(original)) {
		t.Errorf("file changed on a no-op removal:\nwant: %q\ngot:  %q", original, written)
	}
}

// TestEditHostUseCase_Plan_Rebind covers required test 4: --key sets/replaces IdentityFile,
// --unbind removes it, and both set is ErrUsage.
func TestEditHostUseCase_Plan_Rebind(t *testing.T) {
	t.Run("key sets absent IdentityFile", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config")
		writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nHost target\n    HostName x.example.com\n"+sshconfig.ManagedRegionEnd+"\n")

		plan, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "target", Key: "/keys/foo"})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		applier := newHostApplier(dir)
		if _, err := applier.Apply(plan); err != nil {
			t.Fatalf("Apply: %v", err)
		}
		written, _ := os.ReadFile(configPath)
		parsed := sshconfig.Parse(written)
		region := soleMarkedRegion(t, parsed)
		target := soleHostBlock(t, region.Body)
		if got, ok := directiveValue(target, "IdentityFile"); !ok || got != "/keys/foo" {
			t.Errorf("IdentityFile = %q (ok=%v), want /keys/foo", got, ok)
		}
	})

	t.Run("key replaces existing IdentityFile", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config")
		writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nHost target\n    HostName x.example.com\n    IdentityFile /keys/old\n"+sshconfig.ManagedRegionEnd+"\n")

		plan, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "target", Key: "/keys/new"})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		applier := newHostApplier(dir)
		if _, err := applier.Apply(plan); err != nil {
			t.Fatalf("Apply: %v", err)
		}
		written, _ := os.ReadFile(configPath)
		parsed := sshconfig.Parse(written)
		region := soleMarkedRegion(t, parsed)
		target := soleHostBlock(t, region.Body)
		if got, ok := directiveValue(target, "IdentityFile"); !ok || got != "/keys/new" {
			t.Errorf("IdentityFile = %q (ok=%v), want /keys/new", got, ok)
		}
	})

	t.Run("unbind removes IdentityFile", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config")
		writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nHost target\n    HostName x.example.com\n    IdentityFile /keys/old\n"+sshconfig.ManagedRegionEnd+"\n")

		plan, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "target", Unbind: true})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		applier := newHostApplier(dir)
		if _, err := applier.Apply(plan); err != nil {
			t.Fatalf("Apply: %v", err)
		}
		written, _ := os.ReadFile(configPath)
		parsed := sshconfig.Parse(written)
		region := soleMarkedRegion(t, parsed)
		target := soleHostBlock(t, region.Body)
		if _, ok := directiveValue(target, "IdentityFile"); ok {
			t.Error("IdentityFile still present, want it removed")
		}
	})

	t.Run("key and unbind together is ErrUsage", func(t *testing.T) {
		dir := t.TempDir()
		_, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "target", Key: "/keys/x", Unbind: true})
		if !errors.Is(err, ErrUsage) {
			t.Fatalf("Plan error = %v, want ErrUsage", err)
		}
	})
}

// TestEditHostUseCase_Plan_InPlace_CustomGroup covers required test 5: the same directive-change
// shape as test 1, but the stanza lives in ~/.ssh/<group>.sshconfig, not the default region —
// whole-file preservation there too, and ~/.ssh/config itself is completely untouched.
func TestEditHostUseCase_Plan_InPlace_CustomGroup(t *testing.T) {
	dir := t.TempDir()
	groupFile := filepath.Join(dir, "work.sshconfig")
	groupOriginal := "# hasp:owned\n" +
		"Host before\n    User alice\n" +
		"Host target\n    HostName old.example.com\n    User bob\n" +
		"Host after\n    User carol\n"
	writeFile(t, groupFile, groupOriginal)

	configPath := filepath.Join(dir, "config")
	configOriginal := sshconfig.ManagedRegionBegin + "\nInclude " + groupFile + "\n" + sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, configOriginal)

	uc := EditHostUseCase{}
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "target", HostName: strp("new.example.com")})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 1 {
		t.Fatalf("len(Changes) = %d, want 1", len(plan.Changes))
	}
	if len(plan.Witnesses) != 1 || plan.Witnesses[0].Path != groupFile {
		t.Fatalf("Witnesses = %+v, want exactly one on %s", plan.Witnesses, groupFile)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	configAfter, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(configAfter, []byte(configOriginal)) {
		t.Errorf("~/.ssh/config changed, want it completely untouched:\nwant: %q\ngot:  %q", configOriginal, configAfter)
	}

	written, err := os.ReadFile(groupFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(written, []byte("# hasp:owned\n")) {
		t.Errorf("group file header not preserved: %q", written[:min(len(written), 20)])
	}
	parsed := sshconfig.Parse(written)
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
	if findHostBlock(t, parsed.Nodes, "before") == nil {
		t.Error("sibling stanza (before) missing")
	}
	if findHostBlock(t, parsed.Nodes, "after") == nil {
		t.Error("sibling stanza (after) missing")
	}
	target := findHostBlock(t, parsed.Nodes, "target")
	if target == nil {
		t.Fatal("target stanza missing")
	}
	if got, ok := directiveValue(target, "HostName"); !ok || got != "new.example.com" {
		t.Errorf("target HostName = %q (ok=%v), want new.example.com", got, ok)
	}
	if got, ok := directiveValue(target, "User"); !ok || got != "bob" {
		t.Errorf("target User = %q (ok=%v), want bob (untouched)", got, ok)
	}
}

// TestEditHostUseCase_Plan_InPlace_CustomGroup_NoTrailingNewline_AddsDirective covers a fourth
// missing-newline junction found during M3 close-out's own exhaustive audit (not one of Bug A/B/C
// as originally scoped, but the same bug class, caught by the audit's own grep methodology): the
// target stanza is a custom group file's last content with no trailing newline, and the edit adds
// a directive that did not previously exist. applyDirectiveFieldEdits appends the newly-authored
// directive after every preserved one — including, here, the original "User bob" line, whose
// preserved Terminator is nil because it really was the file's very last byte. Before the fix (now
// in HostBlock.render() itself, hostblock.go — the shared rendering primitive, not a call-site
// patch, mirroring MarkedRegion.render()'s own Bug B fix), this glued into "User bob    HostName
// new.example.com", silently corrupting the User directive's own value and hiding the new
// HostName's value inside it.
func TestEditHostUseCase_Plan_InPlace_CustomGroup_NoTrailingNewline_AddsDirective(t *testing.T) {
	dir := t.TempDir()
	groupFile := filepath.Join(dir, "work.sshconfig")
	// No trailing newline: "User bob" is the file's absolute last byte.
	groupOriginal := "# hasp:owned\nHost target\n    User bob"
	writeFile(t, groupFile, groupOriginal)

	configPath := filepath.Join(dir, "config")
	configOriginal := sshconfig.ManagedRegionBegin + "\nInclude " + groupFile + "\n" + sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, configOriginal)

	uc := EditHostUseCase{}
	// HostName was never set on this stanza — this is an addition, not a modification of an
	// existing directive, which is what actually appends past the newline-less last line.
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "target", HostName: strp("new.example.com")})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	written, err := os.ReadFile(groupFile)
	if err != nil {
		t.Fatal(err)
	}
	parsed := sshconfig.Parse(written)
	if !bytes.Equal(sshconfig.RenderNodes(parsed.Nodes), written) {
		t.Error("Render(Parse(written)) != written; round-trip broken")
	}
	target := findHostBlock(t, parsed.Nodes, "target")
	if target == nil {
		t.Fatal("target stanza missing")
	}
	if got, ok := directiveValue(target, "User"); !ok || got != "bob" {
		t.Errorf("target User = %q (ok=%v), want bob (untouched, not glued onto the new directive)", got, ok)
	}
	if got, ok := directiveValue(target, "HostName"); !ok || got != "new.example.com" {
		t.Errorf("target HostName = %q (ok=%v), want new.example.com", got, ok)
	}
}

// TestEditHostUseCase_Plan_CrossGroupMove_DefaultToNewCustom covers required test 6: default group
// -> a brand-new custom group. The Include line lands correctly, the stanza is removed from
// ~/.ssh/config's region, the new group file has the rebuilt stanza (directive change applied),
// and — driving Plan -> Apply -> Derive, mirroring Phase 1's own IncludeStaysDiscoverable
// regression test — the moved host is actually discovered by the read pipeline afterward.
func TestEditHostUseCase_Plan_CrossGroupMove_DefaultToNewCustom(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := sshconfig.ManagedRegionBegin + "\nHost target\n    HostName old.example.com\n" + sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, original)

	uc := EditHostUseCase{}
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "target", HostName: strp("new.example.com"), NewGroup: strp("work")})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2 (this is the merge case: dest creation, then a combined config-path include+removal)", len(plan.Changes))
	}
	workFile := filepath.Join(dir, "work.sshconfig")
	c0, ok := plan.Changes[0].(WriteRegion)
	if !ok || c0.File != workFile {
		t.Fatalf("Changes[0] = %+v, want a WriteRegion targeting %s (destination before source, T20)", plan.Changes[0], workFile)
	}
	c1, ok := plan.Changes[1].(WriteRegion)
	if !ok || c1.File != configPath {
		t.Fatalf("Changes[1] = %+v, want a WriteRegion targeting %s", plan.Changes[1], configPath)
	}
	if len(plan.Witnesses) != 1 || plan.Witnesses[0].Path != configPath {
		t.Fatalf("Witnesses = %+v, want exactly one on %s (the group file's Before is empty, no witness)", plan.Witnesses, configPath)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	workBytes, err := os.ReadFile(workFile)
	if err != nil {
		t.Fatalf("read %s: %v", workFile, err)
	}
	workParsed := sshconfig.Parse(workBytes)
	target := soleHostBlock(t, workParsed.Nodes)
	if got, ok := directiveValue(target, "HostName"); !ok || got != "new.example.com" {
		t.Errorf("moved stanza's HostName = %q (ok=%v), want new.example.com", got, ok)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}
	configParsed := sshconfig.Parse(configBytes)
	if len(configParsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %+v, want none", configParsed.MarkerDefects)
	}
	region := soleMarkedRegion(t, configParsed)
	if findHostBlock(t, region.Body, "target") != nil {
		t.Error("target stanza still present in ~/.ssh/config's region, want it removed")
	}
	var sawInclude bool
	for _, n := range region.Body {
		if d, ok := n.(*sshconfig.Directive); ok && d.Keyword == "Include" {
			if args := d.Args(); len(args) == 1 && args[0] == workFile {
				sawInclude = true
			}
		}
	}
	if !sawInclude {
		t.Errorf("no Include %s line found in the region body: %+v", workFile, region.Body)
	}

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	host := findHostByPattern(m.Hosts, "target")
	if host == nil {
		t.Fatal("moved host not discovered by the read pipeline after Derive")
	}
	resolvedWorkFile, evalErr := filepath.EvalSymlinks(workFile)
	if evalErr != nil {
		resolvedWorkFile = workFile
	}
	if host.HostGroup != resolvedWorkFile {
		t.Errorf("host.HostGroup = %q, want %q", host.HostGroup, resolvedWorkFile)
	}
}

// TestEditHostUseCase_Plan_CrossGroupMove_CustomToExistingDefault covers required test 7: custom
// group -> default group, where the default group already has a region. Removal from the group
// file and correct append into the existing region.
func TestEditHostUseCase_Plan_CrossGroupMove_CustomToExistingDefault(t *testing.T) {
	dir := t.TempDir()
	sourceGroupFile := filepath.Join(dir, "old.sshconfig")
	writeFile(t, sourceGroupFile, "# hasp:owned\nHost target\n    HostName old.example.com\n")

	configPath := filepath.Join(dir, "config")
	original := sshconfig.ManagedRegionBegin + "\nInclude " + sourceGroupFile + "\nHost existing\n    User carol\n" + sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, original)

	uc := EditHostUseCase{}
	empty := ""
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "target", NewGroup: &empty})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2", len(plan.Changes))
	}
	c0, ok := plan.Changes[0].(WriteRegion)
	if !ok || c0.File != configPath {
		t.Fatalf("Changes[0] = %+v, want a WriteRegion targeting %s (destination before source, T20)", plan.Changes[0], configPath)
	}
	c1, ok := plan.Changes[1].(WriteRegion)
	if !ok || c1.File != sourceGroupFile {
		t.Fatalf("Changes[1] = %+v, want a WriteRegion targeting %s", plan.Changes[1], sourceGroupFile)
	}
	if len(plan.Witnesses) != 2 {
		t.Fatalf("len(Witnesses) = %d, want 2", len(plan.Witnesses))
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	configParsed := sshconfig.Parse(configBytes)
	region := soleMarkedRegion(t, configParsed)
	if findHostBlock(t, region.Body, "existing") == nil {
		t.Error("existing stanza missing from the region after the move")
	}
	if findHostBlock(t, region.Body, "target") == nil {
		t.Error("moved (target) stanza missing from the region")
	}

	sourceBytes, err := os.ReadFile(sourceGroupFile)
	if err != nil {
		t.Fatal(err)
	}
	sourceParsed := sshconfig.Parse(sourceBytes)
	if findHostBlock(t, sourceParsed.Nodes, "target") != nil {
		t.Error("target stanza still present in the source group file, want it removed")
	}
}

// TestEditHostUseCase_Plan_CrossGroupMove_CustomToExistingCustom covers required test 8: custom
// group -> a different, pre-existing custom group. Both files updated correctly, ~/.ssh/config
// itself untouched.
func TestEditHostUseCase_Plan_CrossGroupMove_CustomToExistingCustom(t *testing.T) {
	dir := t.TempDir()
	sourceGroupFile := filepath.Join(dir, "old.sshconfig")
	writeFile(t, sourceGroupFile, "# hasp:owned\nHost target\n    HostName old.example.com\n")
	destGroupFile := filepath.Join(dir, "new.sshconfig")
	writeFile(t, destGroupFile, "# hasp:owned\nHost already-there\n    User dave\n")

	configPath := filepath.Join(dir, "config")
	original := sshconfig.ManagedRegionBegin + "\nInclude " + sourceGroupFile + "\nInclude " + destGroupFile + "\n" + sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, original)

	uc := EditHostUseCase{}
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "target", NewGroup: strp("new")})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2", len(plan.Changes))
	}
	c0, ok := plan.Changes[0].(WriteRegion)
	if !ok || c0.File != destGroupFile {
		t.Fatalf("Changes[0] = %+v, want a WriteRegion targeting %s", plan.Changes[0], destGroupFile)
	}
	c1, ok := plan.Changes[1].(WriteRegion)
	if !ok || c1.File != sourceGroupFile {
		t.Fatalf("Changes[1] = %+v, want a WriteRegion targeting %s", plan.Changes[1], sourceGroupFile)
	}
	if len(plan.Witnesses) != 2 {
		t.Fatalf("len(Witnesses) = %d, want 2 (configPath is never written, no witness for it)", len(plan.Witnesses))
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	configAfter, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(configAfter, []byte(original)) {
		t.Errorf("~/.ssh/config changed, want it completely untouched:\nwant: %q\ngot:  %q", original, configAfter)
	}

	destBytes, err := os.ReadFile(destGroupFile)
	if err != nil {
		t.Fatal(err)
	}
	destParsed := sshconfig.Parse(destBytes)
	if findHostBlock(t, destParsed.Nodes, "already-there") == nil {
		t.Error("pre-existing stanza (already-there) missing from the destination group file")
	}
	if findHostBlock(t, destParsed.Nodes, "target") == nil {
		t.Error("moved (target) stanza missing from the destination group file")
	}

	sourceBytes, err := os.ReadFile(sourceGroupFile)
	if err != nil {
		t.Fatal(err)
	}
	sourceParsed := sshconfig.Parse(sourceBytes)
	if findHostBlock(t, sourceParsed.Nodes, "target") != nil {
		t.Error("target stanza still present in the source group file, want it removed")
	}
}

// TestEditHostUseCase_Plan_Combined covers required test 9: directive change + rebind + group
// move, all in one call — all three land correctly together.
func TestEditHostUseCase_Plan_Combined(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := sshconfig.ManagedRegionBegin + "\nHost target\n    HostName old.example.com\n    IdentityFile /old/key\n" + sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, original)

	uc := EditHostUseCase{}
	plan, err := uc.Plan(EditHostRequest{
		KeyDir:   dir,
		Pattern:  "target",
		HostName: strp("new.example.com"),
		Key:      "/new/key",
		NewGroup: strp("combo"),
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	comboFile := filepath.Join(dir, "combo.sshconfig")
	comboBytes, err := os.ReadFile(comboFile)
	if err != nil {
		t.Fatalf("read %s: %v", comboFile, err)
	}
	comboParsed := sshconfig.Parse(comboBytes)
	target := soleHostBlock(t, comboParsed.Nodes)
	if got, ok := directiveValue(target, "HostName"); !ok || got != "new.example.com" {
		t.Errorf("HostName = %q (ok=%v), want new.example.com", got, ok)
	}
	if got, ok := directiveValue(target, "IdentityFile"); !ok || got != "/new/key" {
		t.Errorf("IdentityFile = %q (ok=%v), want /new/key", got, ok)
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	configParsed := sshconfig.Parse(configBytes)
	region := soleMarkedRegion(t, configParsed)
	if findHostBlock(t, region.Body, "target") != nil {
		t.Error("target stanza still present in ~/.ssh/config's region, want it moved out")
	}
}

// TestEditHostUseCase_Plan_ChangeOrdering_ThreeFileMove covers required test 10 (T20): a
// cross-group move creating a new destination group, where source and destination are two
// distinct, pre-existing custom groups and ~/.ssh/config needs a fresh Include line — the worst
// case, three Changes. Plan.Changes must be ordered destination creation, then the config-path
// Include addition, then source removal last.
func TestEditHostUseCase_Plan_ChangeOrdering_ThreeFileMove(t *testing.T) {
	dir := t.TempDir()
	sourceGroupFile := filepath.Join(dir, "source.sshconfig")
	writeFile(t, sourceGroupFile, "# hasp:owned\nHost target\n    HostName old.example.com\n")

	configPath := filepath.Join(dir, "config")
	original := sshconfig.ManagedRegionBegin + "\nInclude " + sourceGroupFile + "\n" + sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, original)

	destGroupFile := filepath.Join(dir, "dest.sshconfig") // does not exist yet

	uc := EditHostUseCase{}
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "target", NewGroup: strp("dest")})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Changes) != 3 {
		t.Fatalf("len(Changes) = %d, want 3", len(plan.Changes))
	}
	var files []string
	for _, c := range plan.Changes {
		wr, ok := c.(WriteRegion)
		if !ok {
			t.Fatalf("Change %+v is not a WriteRegion", c)
		}
		files = append(files, wr.File)
	}
	want := []string{destGroupFile, configPath, sourceGroupFile}
	for i := range want {
		if files[i] != want[i] {
			t.Errorf("Changes[%d].File = %q, want %q (order = %v, want %v)", i, files[i], want[i], files, want)
		}
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := os.Stat(destGroupFile); err != nil {
		t.Errorf("destination group file was not created: %v", err)
	}
}

// TestEditHostUseCase_Plan_NotFound covers required test 11: the pattern matches nothing, managed
// or unmanaged, anywhere reachable.
func TestEditHostUseCase_Plan_NotFound(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, "Host somethingelse\n    HostName x.example.com\n")

	_, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "nosuchhost", HostName: strp("y.example.com")})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
}

// TestEditHostUseCase_Plan_FoundButUnmanaged covers required test 12: the pattern exists as a bare
// top-level stanza in ~/.ssh/config — ErrUsage, with the "use adopt host first" hint.
func TestEditHostUseCase_Plan_FoundButUnmanaged(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, "Host bare\n    HostName bare.example.com\n")

	_, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "bare", HostName: strp("y.example.com")})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
	if !bytes.Contains([]byte(err.Error()), []byte("adopt host first")) {
		t.Errorf("error = %q, want it to hint at adopt host", err.Error())
	}
}

// TestEditHostUseCase_Plan_RejectsNoChanges covers required test 13: an invocation requesting
// nothing is a usage error.
func TestEditHostUseCase_Plan_RejectsNoChanges(t *testing.T) {
	dir := t.TempDir()
	_, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "target"})
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("Plan error = %v, want ErrUsage", err)
	}
	if !bytes.Contains([]byte(err.Error()), []byte("at least one change")) {
		t.Errorf("error = %q, want it to name the missing requirement", err.Error())
	}
}

// TestEditHostUseCase_Plan_FailsClosedOnMarkerDefect covers required test 14 (T18).
func TestEditHostUseCase_Plan_FailsClosedOnMarkerDefect(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	writeFile(t, configPath, sshconfig.ManagedRegionEnd+"\n")

	_, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "anything", HostName: strp("x")})
	if err == nil {
		t.Fatal("Plan succeeded against a malformed marker, want a refusal")
	}
}

// TestEditHostUseCase_Plan_AmbiguousPattern_FirstStanzaWins covers required test 15: two managed
// stanzas share a pattern token — edit host must deterministically target the first one in region
// order (T11's first-obtained-value-wins precedence, mirroring findHostBlockIndex's own documented
// tie-break, exactly as adopt host's own regression test covers on its side).
func TestEditHostUseCase_Plan_AmbiguousPattern_FirstStanzaWins(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	original := sshconfig.ManagedRegionBegin + "\n" +
		"Host shared\n    User first\n" +
		"Host shared\n    User second\n" +
		sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, original)

	uc := EditHostUseCase{}
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "shared", HostName: strp("x.example.com")})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	parsed := sshconfig.Parse(written)
	region := soleMarkedRegion(t, parsed)

	var sharedBlocks []*sshconfig.HostBlock
	for _, n := range region.Body {
		if hb, ok := n.(*sshconfig.HostBlock); ok && len(hb.Patterns) == 1 && hb.Patterns[0] == "shared" {
			sharedBlocks = append(sharedBlocks, hb)
		}
	}
	if len(sharedBlocks) != 2 {
		t.Fatalf("found %d 'shared' blocks, want 2", len(sharedBlocks))
	}
	if got, ok := directiveValue(sharedBlocks[0], "HostName"); !ok || got != "x.example.com" {
		t.Errorf("first block HostName = %q (ok=%v), want x.example.com", got, ok)
	}
	if got, ok := directiveValue(sharedBlocks[0], "User"); !ok || got != "first" {
		t.Errorf("first block User = %q (ok=%v), want first", got, ok)
	}
	if _, ok := directiveValue(sharedBlocks[1], "HostName"); ok {
		t.Error("second block gained a HostName directive, want it untouched")
	}
	if got, ok := directiveValue(sharedBlocks[1], "User"); !ok || got != "second" {
		t.Errorf("second block User = %q (ok=%v), want second (untouched)", got, ok)
	}
}

// TestEditHostUseCase_Plan_WitnessCoverage covers required test 16: every file this use case reads
// current bytes from to build a Change's Before is traced by a Witness — explicitly enumerated
// here across the region-only, group-file-only, and cross-group shapes (each of the tests above
// also asserts this inline at its own call site).
func TestEditHostUseCase_Plan_WitnessCoverage(t *testing.T) {
	t.Run("in-place, default region", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config")
		writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nHost target\n    User bob\n"+sshconfig.ManagedRegionEnd+"\n")

		plan, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "target", User: strp("carol")})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		if len(plan.Witnesses) != 1 || plan.Witnesses[0].Path != configPath {
			t.Fatalf("Witnesses = %+v, want exactly one on %s", plan.Witnesses, configPath)
		}
	})

	t.Run("in-place, custom group", func(t *testing.T) {
		dir := t.TempDir()
		groupFile := filepath.Join(dir, "work.sshconfig")
		writeFile(t, groupFile, "# hasp:owned\nHost target\n    User bob\n")
		configPath := filepath.Join(dir, "config")
		writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nInclude "+groupFile+"\n"+sshconfig.ManagedRegionEnd+"\n")

		plan, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "target", User: strp("carol")})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		if len(plan.Witnesses) != 1 || plan.Witnesses[0].Path != groupFile {
			t.Fatalf("Witnesses = %+v, want exactly one on %s", plan.Witnesses, groupFile)
		}
	})

	t.Run("cross-group, both custom, pre-existing", func(t *testing.T) {
		dir := t.TempDir()
		sourceGroupFile := filepath.Join(dir, "source.sshconfig")
		writeFile(t, sourceGroupFile, "# hasp:owned\nHost target\n    User bob\n")
		destGroupFile := filepath.Join(dir, "dest.sshconfig")
		writeFile(t, destGroupFile, "# hasp:owned\nHost other\n    User dave\n")
		configPath := filepath.Join(dir, "config")
		writeFile(t, configPath, sshconfig.ManagedRegionBegin+"\nInclude "+sourceGroupFile+"\nInclude "+destGroupFile+"\n"+sshconfig.ManagedRegionEnd+"\n")

		plan, err := (EditHostUseCase{}).Plan(EditHostRequest{KeyDir: dir, Pattern: "target", NewGroup: strp("dest")})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		paths := map[string]bool{}
		for _, w := range plan.Witnesses {
			paths[w.Path] = true
		}
		if len(plan.Witnesses) != 2 || !paths[sourceGroupFile] || !paths[destGroupFile] {
			t.Fatalf("Witnesses = %+v, want exactly one on each of %s and %s", plan.Witnesses, sourceGroupFile, destGroupFile)
		}
	})
}

// TestEditHostUseCase_Plan_CrossGroupMove_CustomToExistingCustom_NoTrailingNewline is Bug A's own
// regression test (M3 close-out, this review round): the identical scenario as
// TestEditHostUseCase_Plan_CrossGroupMove_CustomToExistingCustom above, except destGroupFile's last
// byte is not a newline — the shape newhost.go's own withLeadingNewlineIfNeeded fix already handles
// at its "group already exists" call site, but which this cross-group-move destExists branch was
// missing until this fix. Before the fix, the moved stanza's "Host target" header glued directly
// onto "User dave" (destGroupFile's last, newline-less line), producing "User daveHost target" —
// silent, undetectable corruption that also makes "already-there" disappear as its own stanza on
// the very next Parse. This asserts zero MarkerDefects and that both stanzas survive intact and
// separately.
func TestEditHostUseCase_Plan_CrossGroupMove_CustomToExistingCustom_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	sourceGroupFile := filepath.Join(dir, "old.sshconfig")
	writeFile(t, sourceGroupFile, "# hasp:owned\nHost target\n    HostName old.example.com\n")
	destGroupFile := filepath.Join(dir, "new.sshconfig")
	// No trailing newline after "User dave" — the exact shape that glued corrupted the file before
	// this fix.
	writeFile(t, destGroupFile, "# hasp:owned\nHost already-there\n    User dave")

	configPath := filepath.Join(dir, "config")
	original := sshconfig.ManagedRegionBegin + "\nInclude " + sourceGroupFile + "\nInclude " + destGroupFile + "\n" + sshconfig.ManagedRegionEnd + "\n"
	writeFile(t, configPath, original)

	uc := EditHostUseCase{}
	plan, err := uc.Plan(EditHostRequest{KeyDir: dir, Pattern: "target", NewGroup: strp("new")})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	applier := newHostApplier(dir)
	if _, err := applier.Apply(plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	destBytes, err := os.ReadFile(destGroupFile)
	if err != nil {
		t.Fatal(err)
	}
	destParsed := sshconfig.Parse(destBytes)
	if len(destParsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %+v, want none", destParsed.MarkerDefects)
	}
	// The pre-existing separator must be inserted, not glued: "User dave" must still be exactly
	// "dave", not "daveHost target" or similar.
	already := findHostBlock(t, destParsed.Nodes, "already-there")
	if already == nil {
		t.Fatal("pre-existing stanza (already-there) missing from the destination group file")
	}
	assertDirectiveValue(t, already, "User", "dave")
	target := findHostBlock(t, destParsed.Nodes, "target")
	if target == nil {
		t.Fatal("moved (target) stanza missing from the destination group file")
	}
	assertDirectiveValue(t, target, "HostName", "old.example.com")

	sourceBytes, err := os.ReadFile(sourceGroupFile)
	if err != nil {
		t.Fatal(err)
	}
	sourceParsed := sshconfig.Parse(sourceBytes)
	if findHostBlock(t, sourceParsed.Nodes, "target") != nil {
		t.Error("target stanza still present in the source group file, want it removed")
	}
}

// TestEditHostUseCase_Plan_RejectsEmptyRequest covers basic usage-error guards, mirroring
// adopthost_test.go's own.
func TestEditHostUseCase_Plan_RejectsEmptyRequest(t *testing.T) {
	cases := []EditHostRequest{
		{Pattern: "foo", HostName: strp("x")}, // no KeyDir
		{KeyDir: "/tmp", HostName: strp("x")}, // no Pattern
	}
	for i, req := range cases {
		if _, err := (EditHostUseCase{}).Plan(req); !errors.Is(err, ErrUsage) {
			t.Errorf("case %d: err = %v, want ErrUsage", i, err)
		}
	}
}
