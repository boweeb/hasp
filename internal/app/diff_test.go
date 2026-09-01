package app

import (
	"fmt"
	"strings"
	"testing"
)

// TestLineDiff_SmallRegionUnchangedByWindowing is requirement 3 of the windowing fix: a WriteRegion
// whose Before/After is already small (newhost.go's own region-only span, a handful of lines) must
// render identically to lineDiff's pre-windowing output — no elision, because there's nothing
// substantial enough to collapse.
func TestLineDiff_SmallRegionUnchangedByWindowing(t *testing.T) {
	before := []byte("Include foo\nInclude bar\nInclude baz\n")
	after := []byte("Include foo\nInclude qux\nInclude baz\n")

	raw := rawLineDiff(before, after)
	got := lineDiff(before, after)
	if len(got) != len(raw) {
		t.Fatalf("lineDiff produced %d lines, want %d (unwindowed) for a small diff", len(got), len(raw))
	}
	for i := range got {
		if got[i] != raw[i] {
			t.Errorf("line %d = %+v, want %+v (unwindowed)", i, got[i], raw[i])
		}
	}
	for _, l := range got {
		if l.Kind == DiffElided {
			t.Errorf("unexpected DiffElided line in a small diff: %+v", l)
		}
	}
}

// TestLineDiff_DoesNotElideJustOverContextWindow covers requirement 3's explicit example: a gap of
// unchanged lines only slightly longer than the context window (diffContextLines=3) — say 5 lines —
// must not be collapsed. diffMinElidedRun is 7, so a 5-line interior gap is well under threshold and
// must survive untouched.
func TestLineDiff_DoesNotElideJustOverContextWindow(t *testing.T) {
	// 3 lines of context, 1 removed, 5 lines of unchanged context (the gap under test), 1 added, 3
	// lines of context: an interior gap of exactly diffMinElidedRun-2 (5) lines.
	before := []byte("a\nb\nc\nOLD\nd\ne\nf\ng\nh\nx\ny\nz\n")
	after := []byte("a\nb\nc\nd\ne\nf\ng\nh\nNEW\nx\ny\nz\n")

	got := lineDiff(before, after)
	for _, l := range got {
		if l.Kind == DiffElided {
			t.Fatalf("diff = %+v, want no DiffElided line for a 5-line interior gap (below diffMinElidedRun=%d)", got, diffMinElidedRun)
		}
	}
}

// TestLineDiff_ElidesSubstantialInteriorGap confirms an interior gap longer than diffMinElidedRun
// collapses to a single DiffElided marker, keeping diffContextLines lines on each side.
func TestLineDiff_ElidesSubstantialInteriorGap(t *testing.T) {
	var unchanged []string
	for i := 0; i < 20; i++ {
		unchanged = append(unchanged, fmt.Sprintf("line%d", i))
	}
	before := append(append([]string{}, unchanged[:10]...), append([]string{"OLD"}, unchanged[10:]...)...)
	after := append(append([]string{}, unchanged[:10]...), append([]string{"NEW"}, unchanged[10:]...)...)

	got := lineDiff([]byte(strings.Join(before, "\n")+"\n"), []byte(strings.Join(after, "\n")+"\n"))

	var elided []DiffLine
	var context []DiffLine
	for _, l := range got {
		switch l.Kind {
		case DiffElided:
			elided = append(elided, l)
		case DiffContext:
			context = append(context, l)
		}
	}
	if len(elided) != 2 {
		t.Fatalf("elided markers = %d, want 2 (one leading gap, one trailing gap); diff = %+v", len(elided), got)
	}
	// Leading gap: unchanged[:10] has 10 lines, hasBefore=false (start of file), hasAfter=true ->
	// keep the last diffContextLines of it, elide the rest (10 - 3 = 7).
	if elided[0].Text != elidedText(10-diffContextLines) {
		t.Errorf("leading elided marker = %q, want %q", elided[0].Text, elidedText(10-diffContextLines))
	}
	// Trailing gap: unchanged[10:] has 10 lines, hasBefore=true, hasAfter=false (end of file) ->
	// keep the first diffContextLines, elide the rest (10 - 3 = 7).
	if elided[1].Text != elidedText(10-diffContextLines) {
		t.Errorf("trailing elided marker = %q, want %q", elided[1].Text, elidedText(10-diffContextLines))
	}
	if len(context) != 2*diffContextLines {
		t.Errorf("context lines kept = %d, want %d (diffContextLines on each side)", len(context), 2*diffContextLines)
	}
}

// TestLineDiff_ReproducesReviewerFinding_WholeFileWriteRegion reproduces the reviewer's finding
// directly: a ~150-line ~/.ssh/config-shaped file, with adopt host's whole-file Before/After
// (adopthost.go's documented choice), produces an unwindowed diff of essentially the whole file —
// confirming the *unfixed* shape of the bug — and confirms lineDiff's windowed output collapses
// that down to a short, readable diff (a handful of context lines around the actual change, plus
// elision markers for the rest), matching this task's required repro.
func TestLineDiff_ReproducesReviewerFinding_WholeFileWriteRegion(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 75; i++ {
		fmt.Fprintf(&b, "Host server%d\n    HostName server%d.example.com\n", i, i)
	}
	before := []byte(b.String()) // 150 lines

	lines := strings.Split(strings.TrimSuffix(string(before), "\n"), "\n")
	// Relocate one small stanza (as `adopt host` does): insert 2 lines in the middle.
	after := append([]string{}, lines[:40]...)
	after = append(after, "Host relocated", "    HostName relocated.example.com")
	after = append(after, lines[40:]...)
	afterBytes := []byte(strings.Join(after, "\n") + "\n")

	raw := rawLineDiff(before, afterBytes)
	if len(raw) < 140 {
		t.Fatalf("rawLineDiff produced only %d lines, want ~150+ to actually reproduce the reviewer's finding", len(raw))
	}

	windowed := lineDiff(before, afterBytes)
	if len(windowed) >= len(raw) {
		t.Fatalf("windowed diff (%d lines) did not shrink relative to raw (%d lines)", len(windowed), len(raw))
	}
	// A handful of context lines around the 2 added lines, plus exactly 2 elision markers (leading
	// and trailing gap) — nowhere near the ~150 lines the unfixed behavior produces.
	const maxWindowed = 2*diffContextLines*2 + 2 /* added */ + 2 /* elided markers */
	if len(windowed) > maxWindowed {
		t.Errorf("windowed diff = %d lines, want <= %d (short and readable)", len(windowed), maxWindowed)
	}

	var elidedCount int
	for _, l := range windowed {
		if l.Kind == DiffElided {
			elidedCount++
		}
	}
	if elidedCount != 2 {
		t.Errorf("elided markers = %d, want 2 (leading + trailing gap around the one small hunk)", elidedCount)
	}

	t.Logf("raw diff: %d lines; windowed diff: %d lines", len(raw), len(windowed))
}

// TestWindowDiff_ElisionThresholdBoundary pins down windowDiff's exact collapse threshold against
// diffMinElidedRun itself (not a hardcoded literal), so this test breaks loudly if the constant ever
// changes without a boundary re-check. A run of diffMinElidedRun-1 lines must survive untouched; a
// run of exactly diffMinElidedRun lines is the shortest run that does get collapsed (per
// diffMinElidedRun's own doc comment); a run of diffMinElidedRun+1 lines elides one more line than
// that. This exact-boundary case (run length == the constant itself) is what the prior windowing fix
// pass missed.
func TestWindowDiff_ElisionThresholdBoundary(t *testing.T) {
	// buildLines returns diffContextLines lines of leading context, one hunk line (so both
	// hasBefore/hasAfter are true around the run under test), a run of exactly n unchanged context
	// lines, then one more hunk line and diffContextLines lines of trailing context — mirroring an
	// interior gap flanked by hunks on both sides, the shape windowDiff's elision math assumes.
	buildLines := func(n int) []DiffLine {
		var lines []DiffLine
		for i := 0; i < diffContextLines; i++ {
			lines = append(lines, DiffLine{Kind: DiffContext, Text: fmt.Sprintf("lead%d", i)})
		}
		lines = append(lines, DiffLine{Kind: DiffRemoved, Text: "OLD"})
		for i := 0; i < n; i++ {
			lines = append(lines, DiffLine{Kind: DiffContext, Text: fmt.Sprintf("run%d", i)})
		}
		lines = append(lines, DiffLine{Kind: DiffAdded, Text: "NEW"})
		for i := 0; i < diffContextLines; i++ {
			lines = append(lines, DiffLine{Kind: DiffContext, Text: fmt.Sprintf("trail%d", i)})
		}
		return lines
	}
	countElided := func(t *testing.T, lines []DiffLine) (count int, elidedN int) {
		t.Helper()
		for _, l := range lines {
			if l.Kind == DiffElided {
				count++
				var n int
				if l.Text == "... 1 unchanged line ..." {
					n = 1
				} else if _, err := fmt.Sscanf(l.Text, "... %d unchanged lines ...", &n); err != nil {
					t.Fatalf("parse elided count from %q: %v", l.Text, err)
				}
				elidedN = n
			}
		}
		return count, elidedN
	}

	tests := []struct {
		name       string
		n          int
		wantElided bool
		wantN      int
	}{
		{"below threshold: diffMinElidedRun-1 does not elide", diffMinElidedRun - 1, false, 0},
		{"at threshold: diffMinElidedRun elides exactly 1 line", diffMinElidedRun, true, diffMinElidedRun - 2*diffContextLines},
		{"above threshold: diffMinElidedRun+1 elides exactly 2 lines", diffMinElidedRun + 1, true, diffMinElidedRun + 1 - 2*diffContextLines},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := windowDiff(buildLines(tt.n), diffContextLines, diffMinElidedRun)
			count, elidedN := countElided(t, got)
			if tt.wantElided {
				if count != 1 {
					t.Fatalf("run of %d lines: got %d DiffElided markers, want 1; diff = %+v", tt.n, count, got)
				}
				if elidedN != tt.wantN {
					t.Errorf("run of %d lines: elided marker covers %d lines, want %d", tt.n, elidedN, tt.wantN)
				}
			} else if count != 0 {
				t.Fatalf("run of %d lines: got %d DiffElided markers, want 0 (below diffMinElidedRun=%d); diff = %+v", tt.n, count, diffMinElidedRun, got)
			}
		})
	}
}

// TestDiffElided_MarshalsAsDistinctKind confirms DiffElided round-trips through DiffKind's
// String()/MarshalJSON() as its own distinguishable value ("elided"), not silently collapsing to
// "context" — required so a --json consumer (T14) can't mistake a collapse marker for real content.
func TestDiffElided_MarshalsAsDistinctKind(t *testing.T) {
	if got := DiffElided.String(); got != "elided" {
		t.Errorf("DiffElided.String() = %q, want %q", got, "elided")
	}
	data, err := DiffElided.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != `"elided"` {
		t.Errorf("MarshalJSON() = %s, want %q", data, `"elided"`)
	}
}
