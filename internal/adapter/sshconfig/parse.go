package sshconfig

// File is a parsed ssh_config-shaped document.
type File struct {
	Nodes         []Node         // ordered: Line | Directive | HostBlock | MarkedRegion
	MarkerDefects []MarkerDefect // populated at parse time; parsing itself never errors —
	// a malformed marker is data, not a failure (T18, §11)
}

// Parse reads b into a File. Parsing never fails — every byte lands in some node, a recognized
// Directive, an opaque Line, a HostBlock, or a MarkedRegion — which is what makes
// Render(Parse(b)) == b total rather than conditional, outside a marked region (T2).
func Parse(b []byte) *File {
	lines := splitLines(b)
	beginIdx, endIdx, defects := scanMarkers(lines)

	if beginIdx == -1 || endIdx == -1 {
		return &File{Nodes: parseNodes(lines, false), MarkerDefects: defects}
	}

	var nodes []Node
	nodes = append(nodes, parseNodes(lines[:beginIdx], false)...)
	nodes = append(nodes, &MarkedRegion{
		Begin: renderPhysicalLine(lines[beginIdx]),
		End:   renderPhysicalLine(lines[endIdx]),
		Body:  parseNodes(lines[beginIdx+1:endIdx], true),
	})
	nodes = append(nodes, parseNodes(lines[endIdx+1:], false)...)

	return &File{Nodes: nodes, MarkerDefects: defects}
}

// parseNodes classifies a run of physical lines into Nodes and groups HostBlocks. insideRegion
// enables #:hasp metadata-line recognition (§7, T25) — MetadataLine syntax is only meaningful
// inside a MarkedRegion body; outside one, a "#:hasp ..." line is an ordinary comment.
func parseNodes(lines []physicalLine, insideRegion bool) []Node {
	var nodes []Node
	for _, pl := range lines {
		if insideRegion {
			if ml, ok := tryMetadataLine(pl); ok {
				nodes = append(nodes, ml)
				continue
			}
		}
		nodes = append(nodes, classifyLine(pl))
	}
	return groupHostBlocks(nodes)
}

func renderPhysicalLine(pl physicalLine) []byte {
	out := make([]byte, 0, len(pl.Content)+len(pl.Terminator))
	out = append(out, pl.Content...)
	out = append(out, pl.Terminator...)
	return out
}

// physicalLine is one line's content and its terminator, before classification.
type physicalLine struct {
	Content    []byte // not including the terminator
	Terminator []byte // "\n", "\r\n", or nil for a final line with no trailing newline
}

// splitLines splits b into physical lines, preserving CRLF vs LF per line and leaving the final
// line's Terminator nil when b does not end in a newline (T24). An empty b yields zero lines.
func splitLines(b []byte) []physicalLine {
	var lines []physicalLine
	i := 0
	for i < len(b) {
		nlOffset := indexByte(b[i:], '\n')
		if nlOffset == -1 {
			lines = append(lines, physicalLine{Content: b[i:], Terminator: nil})
			break
		}
		pos := i + nlOffset
		if pos > i && b[pos-1] == '\r' {
			lines = append(lines, physicalLine{Content: b[i : pos-1], Terminator: []byte("\r\n")})
		} else {
			lines = append(lines, physicalLine{Content: b[i:pos], Terminator: []byte("\n")})
		}
		i = pos + 1
	}
	return lines
}

func indexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' }

// lineKeyword extracts a line's leading keyword token, if it has one: indentEnd is the offset
// where non-whitespace content begins, keyword is the first token (case preserved), afterKeyword
// is everything following it (separator, value, trailing comment — untouched). ok is false for a
// blank line, a comment-only line, or a line with no recognizable leading token (e.g. one
// starting with '=').
func lineKeyword(content []byte) (keyword string, afterKeyword []byte, indentEnd int, ok bool) {
	for indentEnd < len(content) && isSpace(content[indentEnd]) {
		indentEnd++
	}
	rest := content[indentEnd:]
	if len(rest) == 0 || rest[0] == '#' {
		return "", nil, indentEnd, false
	}
	kwEnd := 0
	for kwEnd < len(rest) && !isSpace(rest[kwEnd]) && rest[kwEnd] != '=' {
		kwEnd++
	}
	if kwEnd == 0 {
		return "", nil, indentEnd, false
	}
	return string(rest[:kwEnd]), rest[kwEnd:], indentEnd, true
}

// classifyLine turns one physical line into a Line (blank, comment, or unrecognized keyword) or
// a Directive (a recognized-keyword line: managedKeywords, or Host/Match — HostBlock grouping,
// groupHostBlocks, re-inspects the latter afterward). A single per-line operation, no lookahead.
func classifyLine(pl physicalLine) Node {
	keyword, afterKeyword, indentEnd, ok := lineKeyword(pl.Content)
	if !ok {
		return &Line{Raw: pl.Content, Terminator: pl.Terminator}
	}
	if !isManagedKeyword(keyword) && !isHostOrMatchKeyword(keyword) {
		return &Line{Raw: pl.Content, Terminator: pl.Terminator}
	}

	valueWithSep, trivia := splitTrailingComment(afterKeyword)

	return &Directive{
		LeadingTrivia: pl.Content[:indentEnd],
		Keyword:       keyword,
		RawValue:      valueWithSep,
		Trivia:        trivia,
		Terminator:    pl.Terminator,
	}
}

// splitTrailingComment finds the first unquoted '#' in b and splits it into the value span
// (with trailing whitespace moved into the comment span) and the trailing-comment span,
// including that whitespace (T17: Trivia includes its leading whitespace). Returns (b, nil) if
// there is no unquoted '#'.
func splitTrailingComment(b []byte) (value, trivia []byte) {
	hashIdx := findUnquotedHash(b)
	if hashIdx == -1 {
		return b, nil
	}
	k := hashIdx
	for k > 0 && isSpace(b[k-1]) {
		k--
	}
	return b[:k], b[k:]
}

// findUnquotedHash returns the index of the first '#' outside a double-quoted span, or -1.
// ssh_config(5): "'#' outside of a quoted string may be used to add a comment to the end of a
// line."
func findUnquotedHash(b []byte) int {
	inQuotes := false
	for i, c := range b {
		switch c {
		case '"':
			inQuotes = !inQuotes
		case '#':
			if !inQuotes {
				return i
			}
		}
	}
	return -1
}

// tokenizeArgs splits a Directive's RawValue into arguments, first skipping the keyword-to-value
// separator (optional whitespace, optional single '=', optional whitespace — ssh_config(5)),
// then splitting on whitespace while honoring double-quoted spans as single tokens whose quote
// characters are stripped (ssh_config(5): "Values may optionally be enclosed in double quotes").
func tokenizeArgs(raw []byte) []string {
	i := 0
	for i < len(raw) && isSpace(raw[i]) {
		i++
	}
	if i < len(raw) && raw[i] == '=' {
		i++
		for i < len(raw) && isSpace(raw[i]) {
			i++
		}
	}
	rest := raw[i:]

	var args []string
	var cur []byte
	inQuotes := false
	started := false

	flush := func() {
		if started {
			args = append(args, string(cur))
			cur = nil
			started = false
		}
	}

	for _, c := range rest {
		switch {
		case c == '"':
			inQuotes = !inQuotes
			started = true // an empty quoted token ("") still produces an empty-string arg
		case isSpace(c) && !inQuotes:
			flush()
		default:
			cur = append(cur, c)
			started = true
		}
	}
	flush()

	return args
}
