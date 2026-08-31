package app

import "strings"

// lineDiff computes a line-oriented diff of before against after via the classic LCS dynamic
// program, returning it as Preview's structured DiffLine form (T26: "WriteRegion computes a real
// line diff of the region's current bytes against its regenerated body"). Region bodies are small
// (a handful to a few dozen lines), so the O(n*m) table this builds is never a real cost — this is
// a preview renderer, not a general-purpose diff library.
func lineDiff(before, after []byte) []DiffLine {
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
