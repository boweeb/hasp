package sshconfig

import (
	"bytes"
	"strings"
	"testing"
)

const wellFormedRegion = "" +
	"# a human comment before the region\n" +
	"# >>> hasp:managed >>>\n" +
	"Include ~/.ssh/work.sshconfig\n" +
	"Include ~/.ssh/personal.sshconfig\n" +
	"# <<< hasp:managed <<<\n" +
	"\n" +
	"Host personal-legacy\n" +
	"    HostName legacy.example.com\n"

func TestRoundTrip_WellFormedRegion(t *testing.T) {
	assertRoundTrip(t, []byte(wellFormedRegion))
}

func TestParse_WellFormedRegion_NoDefects(t *testing.T) {
	f := Parse([]byte(wellFormedRegion))
	if len(f.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %v, want none", f.MarkerDefects)
	}

	var regions int
	for _, n := range f.Nodes {
		if r, ok := n.(*MarkedRegion); ok {
			regions++
			if len(r.Body) != 2 {
				t.Errorf("region body has %d nodes, want 2 Include directives", len(r.Body))
			}
			for _, bn := range r.Body {
				d, ok := bn.(*Directive)
				if !ok || d.Keyword != "Include" {
					t.Errorf("body node = %#v, want an Include Directive", bn)
				}
			}
		}
	}
	if regions != 1 {
		t.Fatalf("got %d MarkedRegion nodes, want 1", regions)
	}
}

func TestRoundTrip_RegionWithMetadataLine(t *testing.T) {
	assertRoundTrip(t, []byte(
		"# >>> hasp:managed >>>\n"+
			"#:hasp created_by = \"new host\"\n"+
			"Include ~/.ssh/work.sshconfig\n"+
			"# <<< hasp:managed <<<\n",
	))
}

func TestParse_MetadataLineRecognizedInsideRegionOnly(t *testing.T) {
	f := Parse([]byte(
		"# >>> hasp:managed >>>\n" +
			"#:hasp created_by = \"new host\"\n" +
			"# <<< hasp:managed <<<\n" +
			"#:hasp not_inside_a_region = true\n",
	))
	var region *MarkedRegion
	for _, n := range f.Nodes {
		if r, ok := n.(*MarkedRegion); ok {
			region = r
		}
	}
	if region == nil {
		t.Fatal("no MarkedRegion found")
	}
	ml, ok := region.Body[0].(*MetadataLine)
	if !ok {
		t.Fatalf("region.Body[0] = %T, want *MetadataLine", region.Body[0])
	}
	if ml.Key != "created_by" || ml.Value != `"new host"` {
		t.Errorf("MetadataLine = %+v, want Key=created_by Value=\"new host\"", ml)
	}

	// Outside the region, the same #:hasp-shaped text is an ordinary opaque Line.
	last := f.Nodes[len(f.Nodes)-1]
	if _, ok := last.(*Line); !ok {
		t.Errorf("outside-region #:hasp line = %T, want *Line (sentinel only recognized inside a region)", last)
	}
}

func TestScanMarkers_UnmatchedBegin(t *testing.T) {
	f := Parse([]byte("# >>> hasp:managed >>>\nInclude ~/.ssh/work.sshconfig\n"))
	if len(f.MarkerDefects) != 1 || f.MarkerDefects[0].Kind != DefectUnmatchedBegin {
		t.Fatalf("MarkerDefects = %v, want exactly one DefectUnmatchedBegin", f.MarkerDefects)
	}
	for _, n := range f.Nodes {
		if _, ok := n.(*MarkedRegion); ok {
			t.Error("no MarkedRegion should be carved when the begin marker never closes")
		}
	}
}

func TestScanMarkers_EndBeforeBegin(t *testing.T) {
	f := Parse([]byte("# <<< hasp:managed <<<\n"))
	if len(f.MarkerDefects) != 1 || f.MarkerDefects[0].Kind != DefectEndBeforeBegin {
		t.Fatalf("MarkerDefects = %v, want exactly one DefectEndBeforeBegin", f.MarkerDefects)
	}
}

func TestScanMarkers_DuplicateBegin(t *testing.T) {
	f := Parse([]byte(
		"# >>> hasp:managed >>>\n" +
			"# >>> hasp:managed >>>\n" +
			"Include ~/.ssh/work.sshconfig\n" +
			"# <<< hasp:managed <<<\n",
	))
	if len(f.MarkerDefects) != 1 || f.MarkerDefects[0].Kind != DefectDuplicateBegin {
		t.Fatalf("MarkerDefects = %v, want exactly one DefectDuplicateBegin", f.MarkerDefects)
	}
	// The first begin still carves a valid region, from the first begin to the end marker.
	var region *MarkedRegion
	for _, n := range f.Nodes {
		if r, ok := n.(*MarkedRegion); ok {
			region = r
		}
	}
	if region == nil {
		t.Fatal("expected a MarkedRegion to still be carved from the first begin marker")
	}
}

func TestScanMarkers_UnmatchedEnd(t *testing.T) {
	f := Parse([]byte(
		"# >>> hasp:managed >>>\n" +
			"Include ~/.ssh/work.sshconfig\n" +
			"# <<< hasp:managed <<<\n" +
			"# <<< hasp:managed <<<\n",
	))
	if len(f.MarkerDefects) != 1 || f.MarkerDefects[0].Kind != DefectUnmatchedEnd {
		t.Fatalf("MarkerDefects = %v, want exactly one DefectUnmatchedEnd", f.MarkerDefects)
	}
}

// TestScanMarkers_DuplicateBeginAfterClose covers the *other* trigger for DefectDuplicateBegin:
// a second begin marker appearing after an earlier region already closed cleanly (not, as
// TestScanMarkers_DuplicateBegin covers, before the first one closes). Both are the same defect
// kind, but the Detail message must describe the actual state, not just the stateOpen case.
func TestScanMarkers_DuplicateBeginAfterClose(t *testing.T) {
	f := Parse([]byte(
		"# >>> hasp:managed >>>\n" +
			"Include ~/.ssh/work.sshconfig\n" +
			"# <<< hasp:managed <<<\n" +
			"# >>> hasp:managed >>>\n",
	))
	if len(f.MarkerDefects) != 1 || f.MarkerDefects[0].Kind != DefectDuplicateBegin {
		t.Fatalf("MarkerDefects = %v, want exactly one DefectDuplicateBegin", f.MarkerDefects)
	}
	if !strings.Contains(f.MarkerDefects[0].Detail, "already closed") {
		t.Errorf("Detail = %q, want it to describe the already-closed case, not the still-open one", f.MarkerDefects[0].Detail)
	}
	// The first, cleanly-closed region must still be carved.
	var region *MarkedRegion
	for _, n := range f.Nodes {
		if r, ok := n.(*MarkedRegion); ok {
			region = r
		}
	}
	if region == nil {
		t.Fatal("expected the first region to still be carved despite the later duplicate begin")
	}
}

func TestScanMarkers_NestedInHostBlock(t *testing.T) {
	f := Parse([]byte(
		"Host work\n" +
			"    HostName example.com\n" +
			"# >>> hasp:managed >>>\n" +
			"Include ~/.ssh/work.sshconfig\n" +
			"# <<< hasp:managed <<<\n",
	))
	if len(f.MarkerDefects) != 2 {
		t.Fatalf("MarkerDefects = %v, want 2 DefectNestedInHostBlock (begin and end)", f.MarkerDefects)
	}
	for _, d := range f.MarkerDefects {
		if d.Kind != DefectNestedInHostBlock {
			t.Errorf("defect kind = %v, want DefectNestedInHostBlock", d.Kind)
		}
	}
	for _, n := range f.Nodes {
		if _, ok := n.(*MarkedRegion); ok {
			t.Error("no MarkedRegion should be carved when both markers are nested inside a Host block")
		}
	}
}

// TestRoundTrip_AllDefectFixtures confirms Render(Parse(b)) == b still holds for every defect
// shape above (§11: reads are fail-open — report every node that parsed cleanly, flag each
// defect by kind and line, but the bytes themselves are never lost).
func TestRoundTrip_AllDefectFixtures(t *testing.T) {
	fixtures := []string{
		"# >>> hasp:managed >>>\nInclude ~/.ssh/work.sshconfig\n",
		"# <<< hasp:managed <<<\n",
		"# >>> hasp:managed >>>\n# >>> hasp:managed >>>\nInclude ~/.ssh/work.sshconfig\n# <<< hasp:managed <<<\n",
		"# >>> hasp:managed >>>\nInclude ~/.ssh/work.sshconfig\n# <<< hasp:managed <<<\n# <<< hasp:managed <<<\n",
		"Host work\n    HostName example.com\n# >>> hasp:managed >>>\nInclude ~/.ssh/work.sshconfig\n# <<< hasp:managed <<<\n",
	}
	for _, fx := range fixtures {
		assertRoundTrip(t, []byte(fx))
	}
}

// TestScanMarkers_HostBlockInsideOpenRegion_NoDefects covers review finding 3: a HostBlock inside
// an open hasp region, followed by a valid end marker, must produce zero MarkerDefects and exactly
// one MarkedRegion recovered with the HostBlock in its Body — the core case the inHostBlock latch's
// own "not armed while state == stateOpen" carve-out (scanMarkers's own doc comment) exists for,
// currently only covered indirectly through internal/app's newhost_test.go fixtures.
func TestScanMarkers_HostBlockInsideOpenRegion_NoDefects(t *testing.T) {
	f := Parse([]byte(
		"# >>> hasp:managed >>>\n" +
			"Host newhost\n" +
			"    User dave\n" +
			"# <<< hasp:managed <<<\n",
	))
	if len(f.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %v, want none", f.MarkerDefects)
	}
	var region *MarkedRegion
	var regions int
	for _, n := range f.Nodes {
		if r, ok := n.(*MarkedRegion); ok {
			regions++
			region = r
		}
	}
	if regions != 1 {
		t.Fatalf("got %d MarkedRegion nodes, want 1", regions)
	}
	if len(region.Body) != 1 {
		t.Fatalf("region body has %d nodes, want 1 HostBlock", len(region.Body))
	}
	hb, ok := region.Body[0].(*HostBlock)
	if !ok {
		t.Fatalf("region.Body[0] = %T, want *HostBlock", region.Body[0])
	}
	if len(hb.Patterns) != 1 || hb.Patterns[0] != "newhost" {
		t.Errorf("Patterns = %v, want [newhost]", hb.Patterns)
	}
}

// TestScanMarkers_RegionBeforePreExistingTopLevelHostBlock covers review finding/bug 2's shape,
// corrected: a hasp-managed region placed immediately *before* a pre-existing top-level Host block
// (rather than appended after it) must parse with zero defects and exactly one MarkedRegion — this
// is the sshconfig-layer fact internal/app's planRegionChange fix (newhost.go) relies on: since the
// region's own begin/end markers appear before scanMarkers's inHostBlock latch is ever armed by
// "Host bastion", the latch never triggers DefectNestedInHostBlock for them, unlike appending the
// region after the block would (region_test.go's own TestScanMarkers_NestedInHostBlock covers that
// failure shape). The pre-existing block itself is preserved outside the region, as ordinary
// top-level content.
func TestScanMarkers_RegionBeforePreExistingTopLevelHostBlock(t *testing.T) {
	f := Parse([]byte(
		"# >>> hasp:managed >>>\n" +
			"Host newhost\n" +
			"    User dave\n" +
			"# <<< hasp:managed <<<\n" +
			"Host bastion\n" +
			"    HostName bastion.example.com\n",
	))
	if len(f.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects = %v, want none", f.MarkerDefects)
	}
	var regions int
	var sawBastion bool
	for _, n := range f.Nodes {
		if _, ok := n.(*MarkedRegion); ok {
			regions++
		}
		if hb, ok := n.(*HostBlock); ok && len(hb.Patterns) == 1 && hb.Patterns[0] == "bastion" {
			sawBastion = true
		}
	}
	if regions != 1 {
		t.Fatalf("got %d MarkedRegion nodes, want 1", regions)
	}
	if !sawBastion {
		t.Error("pre-existing top-level Host bastion block was not preserved outside the region")
	}
}

// TestMarkedRegionRender_InsertsSeparatorBeforeEnd_WhenBodyLacksTrailingNewline is Bug B's own
// isolated unit test, directly against MarkedRegion.render() (via Parse+Render round-tripping the
// rendered bytes, since render() itself is unexported), independent of any app-layer verb — this is
// where the actual fix lives (region.go), not adopthost.go, so it deserves its own coverage at this
// layer, not only indirect coverage through adopthost_test.go's own
// TestAdoptHostUseCase_Plan_NoPriorRegion_NoTrailingNewline.
//
// A region's Body is only ever carved by Parse when End is recognized as its own physical line —
// which already guarantees Body ends in a terminator — so this shape (a Body whose last node has no
// trailing newline right before End) can only arise when app-layer code builds a MarkedRegion
// itself from a node it did not author with an explicit terminator, exactly as adopthost.go's own
// newManagedRegion([]sshconfig.Node{hostBlock}) call does. RenderNodes is exercised here instead of
// render() directly (unexported) — the same public surface app-layer callers actually use.
func TestMarkedRegionRender_InsertsSeparatorBeforeEnd_WhenBodyLacksTrailingNewline(t *testing.T) {
	region := &MarkedRegion{
		Begin: []byte(ManagedRegionBegin + "\n"),
		End:   []byte(ManagedRegionEnd + "\n"),
		Body: []Node{
			&HostBlock{
				Header: &Directive{Keyword: "Host", RawValue: []byte(" adoptme"), Terminator: []byte("\n")},
				Directives: []Node{
					// No trailing newline — the exact shape an already-parsed, file-final HostBlock
					// carries when the file itself has no trailing newline.
					&Directive{LeadingTrivia: []byte("    "), Keyword: "User", RawValue: []byte(" bob"), Terminator: nil},
				},
			},
		},
	}

	rendered := RenderNodes([]Node{region})

	want := ManagedRegionBegin + "\n" + "Host adoptme\n    User bob\n" + ManagedRegionEnd + "\n"
	if string(rendered) != want {
		t.Fatalf("render() = %q, want %q (a separating newline inserted before End)", rendered, want)
	}

	// The inserted separator must make the region recognizable again on the next Parse — the actual
	// consequence of the bug this guards against (DefectUnmatchedBegin, hasp locking itself out of
	// the region it just wrote).
	reparsed := Parse(rendered)
	if len(reparsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects after re-parse = %v, want none", reparsed.MarkerDefects)
	}
	var found *MarkedRegion
	for _, n := range reparsed.Nodes {
		if r, ok := n.(*MarkedRegion); ok {
			found = r
		}
	}
	if found == nil {
		t.Fatal("no MarkedRegion recovered on re-parse")
	}
	if len(found.Body) != 1 {
		t.Fatalf("re-parsed region body has %d nodes, want 1 HostBlock", len(found.Body))
	}

	// Idempotent, per this type's own doc comment (Body's contract is Render(Parse(Render(r))) ==
	// Render(r), not byte-exact for arbitrary body content — D7 elaboration 1). The inserted
	// separator byte is new, never a rewrite of a byte the caller supplied, so this does not touch
	// T2/P2's stricter guarantee, which is scoped to outside a marked region.
	again := reparsed.Render()
	if !bytes.Equal(again, rendered) {
		t.Fatalf("render is not idempotent:\n first: %q\nsecond: %q", rendered, again)
	}
}

// TestMarkedRegionRender_NoTrailingNewlineNeeded_IsNoOp confirms the fix is a true no-op both when
// Body already ends in a newline (the ordinary, Parse-carved case) and when Body is empty — the same
// no-op discipline withLeadingNewlineIfNeeded's own doc comment (internal/app/newhost.go) already
// establishes for junction-type-1's fix, mirrored here for junction-type-2.
func TestMarkedRegionRender_NoTrailingNewlineNeeded_IsNoOp(t *testing.T) {
	t.Run("body already ends in newline", func(t *testing.T) {
		region := &MarkedRegion{
			Begin: []byte(ManagedRegionBegin + "\n"),
			End:   []byte(ManagedRegionEnd + "\n"),
			Body: []Node{
				&Directive{Keyword: "Include", RawValue: []byte(" ~/.ssh/work.sshconfig"), Terminator: []byte("\n")},
			},
		}
		rendered := RenderNodes([]Node{region})
		want := ManagedRegionBegin + "\nInclude ~/.ssh/work.sshconfig\n" + ManagedRegionEnd + "\n"
		if string(rendered) != want {
			t.Fatalf("render() = %q, want %q (no extra separator inserted)", rendered, want)
		}
	})

	t.Run("body is empty", func(t *testing.T) {
		region := &MarkedRegion{Begin: []byte(ManagedRegionBegin + "\n"), End: []byte(ManagedRegionEnd + "\n")}
		rendered := RenderNodes([]Node{region})
		want := ManagedRegionBegin + "\n" + ManagedRegionEnd + "\n"
		if string(rendered) != want {
			t.Fatalf("render() = %q, want %q (empty body inserts nothing)", rendered, want)
		}
	})
}

// TestMarkedRegionRender_InsertsSeparatorBetweenConsecutiveBodyNodes_WhenFirstLacksTrailingNewline
// covers M3 close-out review round 2's finding 2: MarkedRegion.render()'s body loop only guarded
// the final Body->End junction, not the junction between two consecutive Body elements. No current
// app-layer call site ever inserts an untouched, terminator-less node into the *middle* of a
// region's Body (this shape is not producible by Parse or by any of hasp's own write use cases
// today), so it is directly constructed here rather than reached through an app-layer verb — this
// is the hardening the shared appendNodeWithSeparator helper (render.go) exists for, proven to
// actually work, not just to compile.
func TestMarkedRegionRender_InsertsSeparatorBetweenConsecutiveBodyNodes_WhenFirstLacksTrailingNewline(t *testing.T) {
	region := &MarkedRegion{
		Begin: []byte(ManagedRegionBegin + "\n"),
		End:   []byte(ManagedRegionEnd + "\n"),
		Body: []Node{
			// No trailing newline — a node the app layer authored without an explicit terminator,
			// inserted before another Body element rather than as Body's last one.
			&Directive{Keyword: "Include", RawValue: []byte(" ~/.ssh/work.sshconfig"), Terminator: nil},
			&Directive{Keyword: "Include", RawValue: []byte(" ~/.ssh/personal.sshconfig"), Terminator: []byte("\n")},
		},
	}

	rendered := RenderNodes([]Node{region})

	want := ManagedRegionBegin + "\n" +
		"Include ~/.ssh/work.sshconfig\n" +
		"Include ~/.ssh/personal.sshconfig\n" +
		ManagedRegionEnd + "\n"
	if string(rendered) != want {
		t.Fatalf("render() = %q, want %q (a separating newline inserted between the two Body nodes)", rendered, want)
	}

	reparsed := Parse(rendered)
	if len(reparsed.MarkerDefects) != 0 {
		t.Fatalf("MarkerDefects after re-parse = %v, want none", reparsed.MarkerDefects)
	}
	var found *MarkedRegion
	var regions int
	for _, n := range reparsed.Nodes {
		if r, ok := n.(*MarkedRegion); ok {
			regions++
			found = r
		}
	}
	if regions != 1 {
		t.Fatalf("got %d MarkedRegion nodes on re-parse, want 1", regions)
	}
	if len(found.Body) != 2 {
		t.Fatalf("re-parsed region body has %d nodes, want 2 (both Include directives recovered distinctly)", len(found.Body))
	}
	for i, want := range []string{" ~/.ssh/work.sshconfig", " ~/.ssh/personal.sshconfig"} {
		d, ok := found.Body[i].(*Directive)
		if !ok || d.Keyword != "Include" || string(d.RawValue) != want {
			t.Errorf("re-parsed Body[%d] = %#v, want an Include Directive with RawValue %q", i, found.Body[i], want)
		}
	}
}

func TestMarkerDefectKind_String(t *testing.T) {
	cases := map[MarkerDefectKind]string{
		DefectUnmatchedBegin:    "unmatched-begin",
		DefectUnmatchedEnd:      "unmatched-end",
		DefectDuplicateBegin:    "duplicate-begin",
		DefectEndBeforeBegin:    "end-before-begin",
		DefectNestedInHostBlock: "nested-in-host-block",
	}
	for kind, want := range cases {
		if got := kind.String(); got != want {
			t.Errorf("%v.String() = %q, want %q", kind, got, want)
		}
	}
}
