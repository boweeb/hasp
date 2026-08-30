package sshconfig

// File is a parsed ssh_config-shaped document.
type File struct {
	Nodes []Node // ordered: Line | Directive | HostBlock | MarkedRegion
}

// Parse reads b into a File. Parsing never fails — every byte lands in some node, a recognized
// Directive, an opaque Line, or (from Stage 5/6 onward) a HostBlock or MarkedRegion — which is
// what makes Render(Parse(b)) == b total rather than conditional (T2).
func Parse(b []byte) *File {
	var nodes []Node
	for _, ln := range splitLines(b) {
		nodes = append(nodes, classifyLine(ln))
	}
	return &File{Nodes: groupHostBlocks(nodes)}
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

// classifyLine turns one physical line into a Line (blank, comment, or unrecognized keyword) or
// a Directive (a recognized-keyword line, managedKeywords). Host/Match lines are also classified
// as Line here — HostBlock grouping (groupHostBlocks) re-inspects them afterward, so this pass
// stays a single, simple per-line operation with no lookahead.
func classifyLine(pl physicalLine) Node {
	content := pl.Content

	indentEnd := 0
	for indentEnd < len(content) && isSpace(content[indentEnd]) {
		indentEnd++
	}
	rest := content[indentEnd:]

	if len(rest) == 0 || rest[0] == '#' {
		return &Line{Raw: content, Terminator: pl.Terminator}
	}

	kwEnd := 0
	for kwEnd < len(rest) && !isSpace(rest[kwEnd]) && rest[kwEnd] != '=' {
		kwEnd++
	}
	if kwEnd == 0 {
		// Line starts with '=' or similar with no leading keyword token — not a directive shape
		// hasp models; carry it opaquely.
		return &Line{Raw: content, Terminator: pl.Terminator}
	}
	keyword := string(rest[:kwEnd])
	afterKeyword := rest[kwEnd:]

	if !isManagedKeyword(keyword) && !isHostOrMatchKeyword(keyword) {
		return &Line{Raw: content, Terminator: pl.Terminator}
	}

	valueWithSep, trivia := splitTrailingComment(afterKeyword)

	return &Directive{
		LeadingTrivia: content[:indentEnd],
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
