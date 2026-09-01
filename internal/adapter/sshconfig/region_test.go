package sshconfig

import (
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
