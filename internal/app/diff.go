package app

import (
	"fmt"
	"strings"
)

// diffContextLines is how many lines of unchanged DiffContext windowDiff keeps immediately before
// and after each hunk of DiffAdded/DiffRemoved lines — the standard `diff -u`/`git diff` default,
// chosen so a hasp preview reads like the unified diffs every user already knows how to skim
// (P5/D6: preview must be trustworthy and readable, not a wall of unchanged lines).
const diffContextLines = 3

// diffMinElidedRun is the shortest run of unchanged context lines windowDiff will actually
// collapse. Set to more than 2x diffContextLines (i.e. > 6, so 7+) so a run only barely longer than
// the context window on both sides — say 4 or 5 lines — is left exactly as lineDiff produced it
// rather than being replaced by a marker that would save almost nothing; elision only kicks in once
// there's a real, substantial gap to hide (requirement 3 of the windowing fix).
const diffMinElidedRun = 2*diffContextLines + 1

// lineDiff computes a line-oriented diff of before against after via the classic LCS dynamic
// program, then windows the result (windowDiff) so Preview() stays readable regardless of how much
// of before/after is unchanged (T26: "WriteRegion computes a real line diff of the region's current
// bytes against its regenerated body"; P5/D6: preview is not optional, so it must not become
// hundreds of lines of unchanged context on a whole-file WriteRegion like adopt/release host's).
// This is presentation-only: the DiffLine slice returned here feeds Preview().Diff and nothing
// else — Apply never consults it, and before/after themselves are untouched.
func lineDiff(before, after []byte) []DiffLine {
	return windowDiff(rawLineDiff(before, after), diffContextLines, diffMinElidedRun)
}

// rawLineDiff is lineDiff's un-windowed LCS diff — every line of before/after as a DiffLine, with
// no elision. Region bodies (WriteRegion's usual case) are small (a handful to a few dozen lines),
// so the O(n*m) table this builds is never a real cost — this is a preview renderer, not a
// general-purpose diff library.
func rawLineDiff(before, after []byte) []DiffLine {
	a := splitLines(before)
	b := splitLines(after)

	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case a[i] == b[j]:
				lcs[i][j] = lcs[i+1][j+1] + 1
			case lcs[i+1][j] >= lcs[i][j+1]:
				lcs[i][j] = lcs[i+1][j]
			default:
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var out []DiffLine
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			out = append(out, DiffLine{Kind: DiffContext, Text: a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, DiffLine{Kind: DiffRemoved, Text: a[i]})
			i++
		default:
			out = append(out, DiffLine{Kind: DiffAdded, Text: b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, DiffLine{Kind: DiffRemoved, Text: a[i]})
	}
	for ; j < m; j++ {
		out = append(out, DiffLine{Kind: DiffAdded, Text: b[j]})
	}
	return out
}

// splitLines splits b on "\n", dropping exactly one trailing terminator if present, so a
// terminated and unterminated final line don't produce a spurious trailing empty entry.
func splitLines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

// windowDiff collapses long runs of unchanged DiffContext lines in lines down to at most context
// lines kept on each side of every hunk of DiffAdded/DiffRemoved lines, replacing whatever's
// dropped from the middle of a run with a single synthetic DiffElided line — a standard unified-
// diff-style context window (requirement 2 of the windowing fix; T26, P5/D6).
//
// By construction, rawLineDiff never emits two adjacent DiffContext lines split across separate
// runs (each maximal run of context lines is contiguous in lines, and likewise for each maximal run
// of Added/Removed lines), so windowDiff only ever needs to look at whether a given context run is
// preceded and/or followed by a hunk — never at the internal shape of the hunks themselves.
//
// A run shorter than minRun is left untouched, even if it's longer than context on one or both
// sides (requirement 3: don't bother collapsing a handful of lines just because it's slightly more
// than the context window — only elide a real, substantial gap). A run of exactly minRun lines is
// the shortest run that does get collapsed (elided = minRun - frontKeep - backKeep, at least 1
// line). Because minRun is chosen as more than 2*context, this also guarantees frontKeep+backKeep
// never exceeds a run being elided.
func windowDiff(lines []DiffLine, context, minRun int) []DiffLine {
	if len(lines) == 0 {
		return lines
	}

	out := make([]DiffLine, 0, len(lines))
	i := 0
	for i < len(lines) {
		if lines[i].Kind != DiffContext {
			out = append(out, lines[i])
			i++
			continue
		}

		j := i
		for j < len(lines) && lines[j].Kind == DiffContext {
			j++
		}
		run := lines[i:j]
		hasBefore := i > 0         // this run immediately follows a hunk
		hasAfter := j < len(lines) // this run immediately precedes a hunk

		if len(run) < minRun {
			out = append(out, run...)
			i = j
			continue
		}

		frontKeep, backKeep := 0, 0
		if hasBefore {
			frontKeep = context
		}
		if hasAfter {
			backKeep = context
		}

		out = append(out, run[:frontKeep]...)
		elided := len(run) - frontKeep - backKeep
		out = append(out, DiffLine{Kind: DiffElided, Text: elidedText(elided)})
		out = append(out, run[len(run)-backKeep:]...)
		i = j
	}
	return out
}

// elidedText renders windowDiff's collapse-marker text for a run of n omitted unchanged lines.
func elidedText(n int) string {
	if n == 1 {
		return "... 1 unchanged line ..."
	}
	return fmt.Sprintf("... %d unchanged lines ...", n)
}
