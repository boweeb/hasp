// Command docscheck runs the documentation-verification checks this documentation set's own
// culture depends on (docs/tdd.md §17 "The `Docs` target", docs/roadmap.md §5.5 M3.5.3): broken
// markdown links and anchors, every `Dn`/`Tn` citation resolving to a real `<a id>` (and zero `Fn`
// references, since no F-log has ever existed), the decision-log/tech-decision-log per-log
// invariants (anchor IDs contiguous from 1, index row count matching anchor count, every entry
// carrying a `### Consequence`), and citation-anchored verbatim-quotation checking. It is a
// development-time tool, run via `go run ./tools/docscheck` from the repo root and wired as the
// `Docs` Mage target — never imported by hasp itself, and never exec'd by hasp at runtime.
//
// It walks every file matching docs/*.md (non-recursive) and accumulates every finding from all
// four checks before deciding pass/fail, rather than stopping at the first. Two gotchas this tool
// exists to get right, both called out in tdd.md §17: GitHub's anchor-slug algorithm gives each
// space its own hyphen and never collapses repeats, so an em-dash heading (the em-dash rune is
// dropped entirely, leaving two adjacent spaces) yields a double hyphen; and a verbatim quotation
// can wrap across source lines, so comparison must join a paragraph's lines with single spaces
// before matching.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const docsDir = "docs"

// finding is one reported problem, rendered as "path:line: message".
type finding struct {
	path string
	line int
	msg  string
}

func (f finding) String() string {
	return fmt.Sprintf("%s:%d: %s", f.path, f.line, f.msg)
}

// docFile holds one docs/*.md file's content, both raw and split into lines, keyed by base name
// (e.g. "decision-log.md") for cross-file resolution the way links in this corpus are written —
// always a bare filename inside docs/.
type docFile struct {
	path    string // e.g. "docs/decision-log.md"
	base    string // e.g. "decision-log.md"
	content string
	lines   []string
}

func main() {
	files, err := loadDocFiles(docsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "docscheck:", err)
		os.Exit(1)
	}

	byBase := make(map[string]*docFile, len(files))
	for _, f := range files {
		byBase[f.base] = f
	}

	var findings []finding
	findings = append(findings, checkLinks(files, byBase)...)
	findings = append(findings, checkCitations(files, byBase)...)
	findings = append(findings, checkLogInvariants(byBase)...)
	findings = append(findings, checkQuotations(files, byBase)...)

	for _, f := range findings {
		fmt.Println(f.String())
	}
	fmt.Printf("docscheck: %d finding(s)\n", len(findings))
	if len(findings) > 0 {
		os.Exit(1)
	}
}

// loadDocFiles reads every file directly inside dir matching *.md — non-recursive, exactly the
// files the task describes.
func loadDocFiles(dir string) ([]*docFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []*docFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		content := string(b)
		out = append(out, &docFile{
			path:    p,
			base:    e.Name(),
			content: content,
			lines:   strings.Split(content, "\n"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].base < out[j].base })
	return out, nil
}

// ---------------------------------------------------------------------------------------------
// Shared: fenced code blocks, headings, anchors
// ---------------------------------------------------------------------------------------------

var fenceRe = regexp.MustCompile("^\\s*```")

// stripFencedCode blanks out the contents of fenced code blocks, line by line, preserving line
// count and every non-code line untouched. Fenced examples in this corpus include lines that
// look like ATX headings (e.g. a `#:hasp` marker-syntax sample), and letting those leak into
// heading-slug computation would corrupt disambiguation counts for real headings that follow.
func stripFencedCode(lines []string) []string {
	out := make([]string, len(lines))
	inFence := false
	for i, line := range lines {
		if fenceRe.MatchString(line) {
			inFence = !inFence
			out[i] = ""
			continue
		}
		if inFence {
			out[i] = ""
			continue
		}
		out[i] = line
	}
	return out
}

var headingRe = regexp.MustCompile(`^(#{1,6}) (.*)$`)

// slugify implements GitHub's heading-slug algorithm (docs/tdd.md §17's gotcha): lowercase, then
// drop (never replace) every rune that is not a Unicode letter, digit, '-', '_', or literal
// space, then turn each remaining space into a hyphen — with no collapsing of repeats, so an
// em-dash heading's two now-adjacent spaces become a double hyphen.
func slugify(headingText string) string {
	lower := strings.ToLower(headingText)
	var kept strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ' ' {
			kept.WriteRune(r)
		}
	}
	return strings.ReplaceAll(kept.String(), " ", "-")
}

// headingSlugs returns every heading's GitHub slug, in document order, with GitHub's
// disambiguation suffix (-1, -2, ...) applied to repeats.
func headingSlugs(lines []string) []string {
	counts := map[string]int{}
	var slugs []string
	for _, line := range lines {
		m := headingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		base := slugify(m[2])
		n := counts[base]
		counts[base] = n + 1
		if n == 0 {
			slugs = append(slugs, base)
		} else {
			slugs = append(slugs, fmt.Sprintf("%s-%d", base, n))
		}
	}
	return slugs
}

var explicitAnchorRe = regexp.MustCompile(`<a id="([^"]+)"></a>`)

// anchorInfo is a file's resolvable-fragment surface: explicit <a id> tags (exact, case-sensitive)
// and GitHub heading slugs (also case-sensitive, but slugify already lowercases).
type anchorInfo struct {
	ids   map[string]bool
	slugs map[string]bool
}

func buildAnchorInfo(f *docFile) anchorInfo {
	codeless := stripFencedCode(f.lines)
	codelessContent := strings.Join(codeless, "\n")

	ids := map[string]bool{}
	for _, m := range explicitAnchorRe.FindAllStringSubmatch(codelessContent, -1) {
		ids[m[1]] = true
	}
	slugs := map[string]bool{}
	for _, s := range headingSlugs(codeless) {
		slugs[s] = true
	}
	return anchorInfo{ids: ids, slugs: slugs}
}

// ---------------------------------------------------------------------------------------------
// Check A: link/anchor resolution
// ---------------------------------------------------------------------------------------------

var linkRe = regexp.MustCompile(`(!?)\[([^\]]*)\]\(([^)]+)\)`)

func checkLinks(files []*docFile, byBase map[string]*docFile) []finding {
	anchors := make(map[string]anchorInfo, len(files))
	for _, f := range files {
		anchors[f.base] = buildAnchorInfo(f)
	}

	var findings []finding
	for _, f := range files {
		for lineIdx, line := range f.lines {
			lineNo := lineIdx + 1
			for _, m := range linkRe.FindAllStringSubmatch(line, -1) {
				bang, target := m[1], m[3]
				if bang == "!" {
					continue
				}
				if strings.HasPrefix(target, "http://") ||
					strings.HasPrefix(target, "https://") ||
					strings.HasPrefix(target, "mailto:") {
					continue
				}

				filePart, fragPart, hasFrag := target, "", false
				if idx := strings.IndexByte(target, '#'); idx != -1 {
					filePart, fragPart, hasFrag = target[:idx], target[idx+1:], true
				}

				targetBase := filePart
				if targetBase == "" {
					targetBase = f.base
				}

				if filePart != "" {
					if _, ok := byBase[filePart]; !ok {
						findings = append(findings, finding{f.path, lineNo,
							fmt.Sprintf("link target file %q does not exist under docs/", filePart)})
						continue
					}
				}

				if !hasFrag {
					continue
				}
				if fragPart == "" {
					continue
				}

				info, ok := anchors[targetBase]
				if !ok {
					// Should not happen: targetBase either passed the existence check above or
					// is f.base itself, which is always present.
					continue
				}
				if info.ids[fragPart] || info.slugs[fragPart] {
					continue
				}
				findings = append(findings, finding{f.path, lineNo,
					fmt.Sprintf("link fragment %q does not resolve in %s", fragPart, targetBase)})
			}
		}
	}
	return findings
}

// ---------------------------------------------------------------------------------------------
// Check B: Dn/Tn citation resolution, and zero Fn references
// ---------------------------------------------------------------------------------------------
//
// Unlike Check A (which runs stripFencedCode first, since fenced example blocks contain
// heading-shaped lines that would corrupt slug disambiguation), Checks B, C, and D deliberately
// scan raw, unstripped content. tdd.md's fenced Go-snippet blocks embed real Dn/Tn citations in
// their doc comments (e.g. references to D5, D9, D13, T1, T12 inside struct-definition examples),
// and this project's culture wants those kept honest too — so leaving them in view is intentional,
// not an oversight. No fenced block in this corpus currently contains a log heading, a
// "### Consequence" line, or a quotation span, so Checks C and D have no live exposure to this
// asymmetry either way.

var (
	anchorNumRe = regexp.MustCompile(`(?i)<a id="([dt])(\d+)">`)
	citeDRe     = regexp.MustCompile(`\bD(\d+)\b`)
	citeTRe     = regexp.MustCompile(`\bT(\d+)\b`)
	citeFRe     = regexp.MustCompile(`\bF(\d+)\b`)
)

// validAnchorNumbers collects the set of valid N from every <a id="letterN"> in f, matching
// letter case-insensitively (the corpus writes lowercase "d1"/"t1").
func validAnchorNumbers(f *docFile, letter byte) map[int]bool {
	valid := map[int]bool{}
	for _, m := range anchorNumRe.FindAllStringSubmatch(f.content, -1) {
		if strings.EqualFold(m[1], string(letter)) {
			n, err := strconv.Atoi(m[2])
			if err == nil {
				valid[n] = true
			}
		}
	}
	return valid
}

func checkCitations(files []*docFile, byBase map[string]*docFile) []finding {
	var findings []finding

	dLog, hasD := byBase["decision-log.md"]
	tLog, hasT := byBase["tech-decision-log.md"]
	var validD, validT map[int]bool
	if hasD {
		validD = validAnchorNumbers(dLog, 'd')
	}
	if hasT {
		validT = validAnchorNumbers(tLog, 't')
	}

	for _, f := range files {
		for lineIdx, line := range f.lines {
			lineNo := lineIdx + 1

			for _, m := range citeDRe.FindAllStringSubmatch(line, -1) {
				n, _ := strconv.Atoi(m[1])
				if !hasD || !validD[n] {
					findings = append(findings, finding{f.path, lineNo,
						fmt.Sprintf("citation D%d does not resolve to an entry in decision-log.md", n)})
				}
			}
			for _, m := range citeTRe.FindAllStringSubmatch(line, -1) {
				n, _ := strconv.Atoi(m[1])
				if !hasT || !validT[n] {
					findings = append(findings, finding{f.path, lineNo,
						fmt.Sprintf("citation T%d does not resolve to an entry in tech-decision-log.md", n)})
				}
			}
			for _, m := range citeFRe.FindAllStringSubmatch(line, -1) {
				findings = append(findings, finding{f.path, lineNo,
					fmt.Sprintf("F-log reference %q found; no F-log has ever existed in this project", m[0])})
			}
		}
	}
	return findings
}

// ---------------------------------------------------------------------------------------------
// Check C: per-log invariants (decision-log.md / D, tech-decision-log.md / T)
// ---------------------------------------------------------------------------------------------

func checkLogInvariants(byBase map[string]*docFile) []finding {
	var findings []finding
	if f, ok := byBase["decision-log.md"]; ok {
		findings = append(findings, checkOneLog(f, "D")...)
	}
	if f, ok := byBase["tech-decision-log.md"]; ok {
		findings = append(findings, checkOneLog(f, "T")...)
	}
	return findings
}

func checkOneLog(f *docFile, prefix string) []finding {
	var findings []finding

	anchorLineRe := regexp.MustCompile(`(?i)<a id="` + strings.ToLower(prefix) + `(\d+)">`)
	entryHeadingRe := regexp.MustCompile(`^## ` + prefix + `(\d+) `)
	consequenceRe := regexp.MustCompile(`^### Consequence`)
	indexHeadingRe := regexp.MustCompile(`^## `)
	indexRowRe := regexp.MustCompile(`^\|\s*\[` + prefix + `\d+\]\(#` + strings.ToLower(prefix) + `\d+\)`)

	// 1. Anchor numbers: exactly {1, ..., max}, no gaps, no duplicates.
	type anchorHit struct {
		n    int
		line int
	}
	var anchorHits []anchorHit
	for lineIdx, line := range f.lines {
		m := anchorLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		anchorHits = append(anchorHits, anchorHit{n, lineIdx + 1})
	}

	seen := map[int]int{} // n -> count
	maxN := 0
	for _, h := range anchorHits {
		seen[h.n]++
		if h.n > maxN {
			maxN = h.n
		}
	}
	for n, count := range seen {
		if count > 1 {
			findings = append(findings, finding{f.path, 0,
				fmt.Sprintf("%s%d anchor appears %d times (duplicate)", prefix, n, count)})
		}
	}
	for n := 1; n <= maxN; n++ {
		if seen[n] == 0 {
			findings = append(findings, finding{f.path, 0,
				fmt.Sprintf("%s%d anchor is missing (gap in %s1..%s%d)", prefix, n, prefix, prefix, maxN)})
		}
	}
	numAnchors := len(anchorHits)

	// 2. Index table row count == anchor count.
	inIndex := false
	numIndexRows := 0
	for _, line := range f.lines {
		trimmed := strings.TrimRight(line, " \t")
		if !inIndex {
			if trimmed == "## Index" {
				inIndex = true
			}
			continue
		}
		if indexHeadingRe.MatchString(line) {
			break
		}
		if indexRowRe.MatchString(line) {
			numIndexRows++
		}
	}
	if numIndexRows != numAnchors {
		findings = append(findings, finding{f.path, 0,
			fmt.Sprintf("## Index has %d row(s) for %s but %d anchor(s) exist", numIndexRows, prefix, numAnchors)})
	}

	// 3. Every entry (span from "## <prefix>N " heading to the next such heading, or EOF)
	// contains a line starting with "### Consequence" (covers the "### Consequence if accepted"
	// variant too — a prefix match is intentional).
	type entrySpan struct {
		n         int
		startLine int // 0-indexed
		endLine   int // exclusive, 0-indexed
	}
	var entryHeadingLines []int
	var entryNums []int
	for lineIdx, line := range f.lines {
		m := entryHeadingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		entryHeadingLines = append(entryHeadingLines, lineIdx)
		entryNums = append(entryNums, n)
	}
	var spans []entrySpan
	for i, startLine := range entryHeadingLines {
		endLine := len(f.lines)
		if i+1 < len(entryHeadingLines) {
			endLine = entryHeadingLines[i+1]
		}
		spans = append(spans, entrySpan{n: entryNums[i], startLine: startLine, endLine: endLine})
	}
	for _, span := range spans {
		hasConsequence := false
		for _, line := range f.lines[span.startLine:span.endLine] {
			if consequenceRe.MatchString(line) {
				hasConsequence = true
				break
			}
		}
		if !hasConsequence {
			findings = append(findings, finding{f.path, span.startLine + 1,
				fmt.Sprintf("%s%d has no line starting with \"### Consequence\"", prefix, span.n)})
		}
	}

	return findings
}

// ---------------------------------------------------------------------------------------------
// Check D: verbatim quotation checking
// ---------------------------------------------------------------------------------------------

var (
	// project-assessment-2026-08.md is included alongside the other four cross-referenced docs:
	// tech-decision-log.md's T27 links and quotes it directly (docs/project-assessment-2026-08.md
	// §6's Phase 1/2 text), so it is a real, checkable citation target in this corpus too.
	citeLinkRe   = regexp.MustCompile(`\]\((design\.md|tdd\.md|roadmap\.md|decision-log\.md|tech-decision-log\.md|project-assessment-2026-08\.md)(?:#[^)]*)?\)`)
	citeBacktick = regexp.MustCompile("`(design\\.md|tdd\\.md|roadmap\\.md|decision-log\\.md|tech-decision-log\\.md|project-assessment-2026-08\\.md)`")
	citeBareD    = regexp.MustCompile(`\bD(\d+)\b`)
	citeBareT    = regexp.MustCompile(`\bT(\d+)\b`)
	citeBareP    = regexp.MustCompile(`\bP\d+\b`)
	citeBareJ    = regexp.MustCompile(`\bJ\d+\b`)

	quoteDoubleRe = regexp.MustCompile(`\*\*"([^"]*)"\*\*`)
	quoteSingleRe = regexp.MustCompile(`\*"([^"]*)"\*`)
)

// paragraph is a blank-line-delimited block of a doc file, with its lines joined by single spaces
// (required: a quotation can visually wrap across two source lines, and comparison must not be
// fooled by that — tdd.md §17's own gotcha). lineAt maps a byte offset in text back to the
// original source line it came from, for reporting.
type paragraph struct {
	text   string
	lineAt []int
}

func splitParagraphs(f *docFile) []paragraph {
	var paragraphs []paragraph
	var curLines []string
	var curLineNums []int

	flush := func() {
		if len(curLines) == 0 {
			return
		}
		var sb strings.Builder
		var lineAt []int
		for i, l := range curLines {
			if i > 0 {
				sb.WriteByte(' ')
				lineAt = append(lineAt, curLineNums[i])
			}
			sb.WriteString(l)
			for j := 0; j < len(l); j++ {
				lineAt = append(lineAt, curLineNums[i])
			}
		}
		paragraphs = append(paragraphs, paragraph{text: sb.String(), lineAt: lineAt})
		curLines = nil
		curLineNums = nil
	}

	for lineIdx, line := range f.lines {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		curLines = append(curLines, line)
		curLineNums = append(curLineNums, lineIdx+1)
	}
	flush()
	return paragraphs
}

// quoteSpan is one verbatim-quotation match within a paragraph's joined text.
type quoteSpan struct {
	start, end int
	quoted     string
}

// findQuotes locates every *"..."* / **"..."** quotation span in text, matching asterisk counts
// on both sides (Go's RE2-based regexp has no backreference support, so this is done as two
// passes instead of the single backreferenced regex a PCRE-style engine would use: find
// double-asterisk spans first, then single-asterisk spans that do not overlap one already found).
func findQuotes(text string) []quoteSpan {
	var spans []quoteSpan
	covered := make([]bool, len(text))

	for _, m := range quoteDoubleRe.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[0], m[1]
		spans = append(spans, quoteSpan{start: start, end: end, quoted: text[m[2]:m[3]]})
		for i := start; i < end; i++ {
			covered[i] = true
		}
	}
	for _, m := range quoteSingleRe.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[0], m[1]
		overlap := false
		for i := start; i < end; i++ {
			if covered[i] {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}
		spans = append(spans, quoteSpan{start: start, end: end, quoted: text[m[2]:m[3]]})
	}

	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	return spans
}

// pivotBoundaryRe treats a clause-pivoting em dash as a citation-search boundary, even when
// immediately followed by closing markdown emphasis (`**`, `*`, “ ` “) or a closing quote mark
// before the whitespace, e.g. "...text— *next*". This corpus routinely uses "— " to pivot to a new
// clause with its own subject inside one grammatical sentence — most often introducing a verbatim
// quotation from a source this tool cannot check (a third-party README, a stdlib doc comment,
// project-assessment-2026-08.md's own later text), immediately after an unrelated citation that
// only supported the clause before the pivot. A colon is deliberately NOT treated as a boundary,
// unlike em dash: this corpus's most common way to introduce a genuine, checkable citation is
// "`design.md` §9 states it directly: *"..."*", and a colon boundary would blind the check to
// drift in exactly that pattern.
//
// Sentence-ending punctuation (./!/?) is deliberately NOT a boundary, despite an earlier version of
// this tool treating it as one: this corpus routinely writes a short citation-only sentence
// ("Full argument: [T14](tech-decision-log.md#t14).") immediately followed by the sentence that
// actually introduces the quotation ("§6.3 requires *"..."*"), and a period boundary there strands
// the citation outside the search window. A verifier pass proved this concretely: corrupting the
// quoted word in exactly that tdd.md passage (§6.3's "*"every read has a machine-readable form"*")
// went completely undetected under the period-boundary rule, since the window narrowed to text
// containing no citation at all. Paragraph-wide search modulo the em-dash pivot above and
// sealedByParen below was checked against every quotation in this corpus and produces no misfire.
var pivotBoundaryRe = regexp.MustCompile("—[*`\"']*\\s")

// sentenceWindow narrows a citation search from "everywhere earlier in the paragraph" down to
// "since the last em-dash pivot" (see pivotBoundaryRe) — a narrower cut than tdd.md §17's literal
// "everything in the paragraph before this quote" only for the one pattern this corpus
// demonstrably uses to jump to an unrelated, unattributable quotation mid-sentence.
func sentenceWindow(before string) string {
	locs := pivotBoundaryRe.FindAllStringIndex(before, -1)
	if len(locs) == 0 {
		return before
	}
	last := locs[len(locs)-1]
	return before[last[1]:]
}

// closedParenSpan is a matched (opening index, closing index) pair of parentheses found in a
// citation search window, closing index inclusive of the ')' itself.
type closedParenSpan struct{ open, close int }

// closedParenSpans finds every fully-matched "(...)" span in s via a paren-depth stack. An
// unmatched trailing "(" (still open at the end of s) is not included.
func closedParenSpans(s string) []closedParenSpan {
	var stack []int
	var spans []closedParenSpan
	for i, r := range s {
		switch r {
		case '(':
			stack = append(stack, i)
		case ')':
			if len(stack) == 0 {
				continue
			}
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			spans = append(spans, closedParenSpan{open: open, close: i})
		}
	}
	return spans
}

// sealedByParen reports whether [start, end) sits entirely inside a "(...)" span that has already
// closed by the end of the search window — i.e. the citation is its own self-contained
// parenthetical aside, not the introduction to whatever clause follows the closing paren.
func sealedByParen(spans []closedParenSpan, start, end int) bool {
	for _, sp := range spans {
		if sp.open <= start && end <= sp.close {
			return true
		}
	}
	return false
}

// findCitationSource looks for the six citation patterns (a markdown link to one of the five
// docs, a bare backtick-quoted filename, or a bare Dn/Tn/Pn/Jn token) within before, and returns
// the source file named by whichever match ends closest to (nearest before) the quotation. It
// returns ok=false if none of the patterns appear anywhere in before.
//
// A match fully sealed inside its own already-closed "(...)" — e.g. "...requires ([T23](...)):
// `x/crypto/ssh`..." — is excluded: it is a self-contained parenthetical aside, not the citation
// introducing whatever clause follows it. This is a judgment call beyond tdd.md §17's literal
// text, needed because this corpus places exactly that shape immediately before a verbatim quote
// from a source this tool cannot check (a stdlib doc comment), and the parenthetical citation
// would otherwise misattribute it.
func findCitationSource(before string) (source string, ok bool) {
	spans := closedParenSpans(before)
	bestEnd := -1

	consider := func(start, end int, src string) {
		if sealedByParen(spans, start, end) {
			return
		}
		if end > bestEnd {
			bestEnd = end
			source = src
		}
	}

	for _, m := range citeLinkRe.FindAllStringSubmatchIndex(before, -1) {
		consider(m[0], m[1], before[m[2]:m[3]])
	}
	for _, m := range citeBacktick.FindAllStringSubmatchIndex(before, -1) {
		consider(m[0], m[1], before[m[2]:m[3]])
	}
	for _, m := range citeBareD.FindAllStringSubmatchIndex(before, -1) {
		consider(m[0], m[1], "decision-log.md")
	}
	for _, m := range citeBareT.FindAllStringSubmatchIndex(before, -1) {
		consider(m[0], m[1], "tech-decision-log.md")
	}
	for _, m := range citeBareP.FindAllStringIndex(before, -1) {
		consider(m[0], m[1], "design.md")
	}
	for _, m := range citeBareJ.FindAllStringIndex(before, -1) {
		consider(m[0], m[1], "design.md")
	}

	if bestEnd == -1 {
		return "", false
	}
	return source, true
}

// wordNormalize reduces s to a lowercased, whitespace-collapsed sequence of words, with every
// character that is not a Unicode letter or digit dropped (hyphens and underscores become a
// space instead, so a hyphenated compound still separates as two words) — the comparison tdd.md
// §17 requires so a quotation that wraps across source lines is still recognized (whitespace
// collapsing), plus two further judgment calls beyond §17's literal "strip * and `" text, both
// needed because this corpus's own quoting convention routinely re-punctuates and re-cases a
// quotation to fit the citing sentence's grammar rather than preserving the source's exact marks:
//
//   - Case-folding: a quotation's leading letter is routinely re-cased to fit mid-sentence use
//     (e.g. "Decide the store" quoted as "decide the store") — the same kind of cosmetic variance
//     P9/J2's own "punctuation and case shouldn't matter" principle already treats as
//     insignificant elsewhere in this project, not the substantive word-level drift this check
//     exists to catch.
//   - Punctuation-stripping beyond `*`/“ ` “: a quotation's internal and trailing punctuation is
//     routinely adapted (a source's period becomes the quoting sentence's comma or semicolon) when
//     a fragment is spliced into surrounding prose. Comparing word sequences with all punctuation
//     removed, rather than requiring identical marks at the exact same positions, absorbs that
//     without weakening the check against real substantive drift (wrong or reordered words).
func wordNormalize(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '-' || r == '_' || unicode.IsSpace(r):
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// quoteMatchesSource asserts that quote (the raw captured text between the quote marks) appears
// in source (the source file's full raw content) once both are word-normalized. A quotation may
// splice two non-adjacent spans of the same source together with a literal "..." — eliding the
// material in between rather than quoting it — so quote is split on "..." first, and each
// resulting segment is required to appear in source, in order, rather than requiring the whole
// quote to be one contiguous span.
func quoteMatchesSource(quote, source string) bool {
	normSource := wordNormalize(source)
	pos := 0
	foundAny := false
	for _, seg := range strings.Split(quote, "...") {
		normSeg := wordNormalize(seg)
		if normSeg == "" {
			continue
		}
		foundAny = true
		idx := strings.Index(normSource[pos:], normSeg)
		if idx == -1 {
			return false
		}
		pos += idx + len(normSeg)
	}
	return foundAny
}

// knownHistoricalQuote is a citation-anchored quote this tool cannot verify for a reason inherent
// to the append-only log convention itself, not a checker bug: the quotation is of a passage that
// was verbatim-true in design.md/tdd.md when the citing entry was written, but has since been
// amended there by a later, entirely legitimate decision — and the citing entry says so itself.
// Append-only means the entry is never rewritten to track the amendment; docs/README.md's own
// rule is explicit that the log is deliberately allowed to diverge from the doc it once quoted
// ("the doc is the truth; the log is the history"). This is intentionally a narrow, exact-text
// allowlist (not a general exemption for historical quotes), so a future, genuinely new drift
// elsewhere is never silently swallowed by it.
type knownHistoricalQuote struct {
	file   string
	quoted string
}

var knownHistoricalQuotes = []knownHistoricalQuote{
	{
		// tech-decision-log.md T7, dated 2026-08-28, quotes P9 as it read that day. D16
		// (2026-08-29, "ratified upstream" from this same entry per T7's own header) amended
		// P9's final clause the very next day. T7's own next paragraph says so in so many words:
		// "That is P9 as it stood when this entry was written. ... The quotation above is
		// preserved as the text this entry actually reasoned against." Editing the quote to
		// chase design.md's current text would falsify the historical record T7 is deliberately
		// keeping; leaving tech-decision-log.md untouched is exactly what M3.5.3's own scope
		// restricts this task to.
		file:   "tech-decision-log.md",
		quoted: "Only as a last resort, a hasp settings file — and no case has yet required one.",
	},
	{
		// decision-log.md D16 itself (dated 2026-08-29) is the entry that amends P9's final
		// clause, and quotes P9 as it read the moment before that amendment took effect — the very
		// same historical clause T7 above preserves, from the other side of the same event. D16's
		// own next sentence names the amendment explicitly ("docs/design.md §4 states the amended
		// form"), so this is not a checker-missed drift: design.md's current P9 (§4) no longer
		// carries this clause on purpose.
		file:   "decision-log.md",
		quoted: "No case has yet required one",
	},
	{
		// tech-decision-log.md T27 (dated 2026-08-29) quotes design.md §10's "implementation
		// stack" deferral item as it read before this same entry closed it — T27's own sentence
		// says so directly: "Before this entry closed it, the item read: ...". design.md §10 no
		// longer carries a deferred implementation-stack item because T27 is exactly the decision
		// that resolved it (Go, per docs/tdd.md §2), so its absence from the current doc is the
		// intended outcome T27 records, not drift.
		file:   "tech-decision-log.md",
		quoted: "**Implementation stack** — language, runtime, distribution, dependencies. Nothing in this document assumes any of them.",
	},
}

func isKnownHistoricalQuote(file, quoted string) bool {
	for _, k := range knownHistoricalQuotes {
		if k.file == file && k.quoted == quoted {
			return true
		}
	}
	return false
}

func checkQuotations(files []*docFile, byBase map[string]*docFile) []finding {
	var findings []finding
	for _, f := range files {
		for _, para := range splitParagraphs(f) {
			for _, q := range findQuotes(para.text) {
				before := sentenceWindow(para.text[:q.start])
				source, ok := findCitationSource(before)
				if !ok {
					continue // unattributed/external quote: not this tool's to check
				}
				srcFile, haveSource := byBase[source]
				if !haveSource {
					continue // the named source file doesn't exist in docs/; Check A covers that
				}
				if wordNormalize(q.quoted) == "" || quoteMatchesSource(q.quoted, srcFile.content) {
					continue
				}
				if isKnownHistoricalQuote(f.base, q.quoted) {
					continue
				}
				lineNo := 1
				if q.start < len(para.lineAt) {
					lineNo = para.lineAt[q.start]
				} else if len(para.lineAt) > 0 {
					lineNo = para.lineAt[len(para.lineAt)-1]
				}
				findings = append(findings, finding{f.path, lineNo,
					fmt.Sprintf("quotation %q attributed to %s does not appear verbatim there", q.quoted, source)})
			}
		}
	}
	return findings
}
