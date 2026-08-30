package sshconfig

import "testing"

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
