// Command docscheck runs the documentation-verification checks this documentation set's own
// culture depends on (docs/tdd.md §17 "The `Docs` target", docs/roadmap.md §5.5 M3.5.3): broken
// markdown links and anchors, every `Dn`/`Tn` citation resolving to a real decision file (and zero
// `Fn` references, since no F-log has ever existed), the per-namespace decision invariants
// (contiguous IDs from 1, front matter agreeing with the filename, every entry carrying a
// `### Consequence`), and citation-anchored verbatim-quotation checking. It is a development-time
// tool, run via `go run ./tools/docscheck` from the repo root and wired as the `Docs` Mage target —
// never imported by hasp itself, and never exec'd by hasp at runtime.
//
// The decision logs live one entry per file at backlog/decisions/decision-{D,T}<n>.md (T56); the
// docs/decision-log.md and docs/tech-decision-log.md stubs keep only the rules. Scope differs by
// check, deliberately:
//
//   - Check A (links) reads docs/*.md, backlog/decisions, the backlog task/draft/milestone/doc/
//     completed folders, and the root *.md files, resolving every link relative to the file it
//     appears in.
//   - Check B (citations) reads the same set, so a task citing a nonexistent T99 is caught. Its
//     zero-F-refs rule exempts exactly the UAT finding ID shape `U<n>.F<m>` (tasks carry those).
//   - Check C (invariants) reads backlog/decisions only.
//   - Check D (quotations) reads docs/*.md and backlog/decisions only: backlog tasks quote text
//     from outside the repository (the UAT log), which this tool cannot and should not verify.
//
// It accumulates every finding from every check before deciding pass/fail, rather than
// stopping at the first. Two gotchas this tool exists to get right, both called out in tdd.md §17:
// GitHub's anchor-slug algorithm gives each space its own hyphen and never collapses repeats, so an
// em-dash heading (the em-dash rune is dropped entirely, leaving two adjacent spaces) yields a
// double hyphen; and a verbatim quotation can wrap across source lines, so comparison must join a
// paragraph's lines with single spaces before matching.
package main

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	docsDir      = "docs"
	decisionsDir = "backlog/decisions"
)

// scanDirs are the directories (relative to the repo root, non-recursive) whose *.md files are
// loaded; "." is the root, for README.md, SECURITY.md, CLAUDE.md, and AGENTS.md.
var scanDirs = []string{
	".",
	docsDir,
	decisionsDir,
	"backlog/tasks",
	"backlog/drafts",
	"backlog/milestones",
	"backlog/docs",
	"backlog/completed",
}

// finding is one reported problem, rendered as "path:line: message".
type finding struct {
	path string
	line int
	msg  string
}

func (f finding) String() string {
	return fmt.Sprintf("%s:%d: %s", f.path, f.line, f.msg)
}

// docFile holds one markdown file's content, both raw and split into lines, keyed by its
// slash-separated path relative to the repo root (e.g. "docs/design.md") for cross-file resolution.
type docFile struct {
	path    string // e.g. "docs/decision-log.md"
	base    string // e.g. "decision-log.md"
	content string
	lines   []string
}

func (f *docFile) dir() string { return path.Dir(f.path) }

// inQuoteScope reports whether Check D reads this file (see the package doc).
func (f *docFile) inQuoteScope() bool {
	d := f.dir()
	return d == docsDir || d == decisionsDir
}

func main() {
	findings, err := run(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "docscheck:", err)
		os.Exit(1)
	}
	for _, f := range findings {
		fmt.Println(f.String())
	}
	fmt.Printf("docscheck: %d finding(s)\n", len(findings))
	if len(findings) > 0 {
		os.Exit(1)
	}
}

// run loads every in-scope file under root and returns the findings of every check.
func run(root string) ([]finding, error) {
	files, err := loadFiles(root)
	if err != nil {
		return nil, err
	}
	byPath := make(map[string]*docFile, len(files))
	for _, f := range files {
		byPath[f.path] = f
	}

	var findings []finding
	findings = append(findings, checkLinks(root, files, byPath)...)
	findings = append(findings, checkCitations(files, byPath)...)
	findings = append(findings, checkLogInvariants(byPath)...)
	findings = append(findings, checkQuotations(files, buildSources(byPath))...)
	return findings, nil
}

// loadFiles reads every *.md file directly inside each scanDirs entry (non-recursive). A missing
// directory is skipped: backlog/tasks and friends need not exist.
func loadFiles(root string) ([]*docFile, error) {
	var out []*docFile
	for _, dir := range scanDirs {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			rel := path.Join(dir, e.Name())
			b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				return nil, err
			}
			content := string(b)
			out = append(out, &docFile{
				path:    rel,
				base:    e.Name(),
				content: content,
				lines:   strings.Split(content, "\n"),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
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

// linkRe matches inline links. A target may be wrapped in angle brackets (`[x](<a b.md>)`, the
// CommonMark form for a target containing spaces); the brackets are stripped before resolution.
// A link title (`[x](url "title")`) is not supported: this corpus uses none.
var linkRe = regexp.MustCompile(`(!?)\[([^\]]*)\]\((<[^>]*>|[^)]+)\)`)

// checkLinks resolves every inline markdown link relative to the file it appears in. A target that
// is a loaded markdown file gets full fragment checking; a target that exists on disk but is not
// loaded (a directory, a Go file, a generated docs/cli page) is only checked for existence.
func checkLinks(root string, files []*docFile, byPath map[string]*docFile) []finding {
	anchors := make(map[string]anchorInfo, len(files))
	for _, f := range files {
		anchors[f.path] = buildAnchorInfo(f)
	}

	var findings []finding
	for _, f := range files {
		for lineIdx, line := range f.lines {
			lineNo := lineIdx + 1
			for _, m := range linkRe.FindAllStringSubmatch(line, -1) {
				bang, target := m[1], strings.TrimSuffix(strings.TrimPrefix(m[3], "<"), ">")
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

				targetPath := f.path
				if filePart != "" {
					// Backlog task files have spaces in their names, which links write as %20.
					if un, err := url.PathUnescape(filePart); err == nil {
						filePart = un
					}
					targetPath = path.Join(f.dir(), filePart)
					if _, ok := byPath[targetPath]; !ok {
						if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(targetPath))); err != nil {
							findings = append(findings, finding{f.path, lineNo,
								fmt.Sprintf("link target file %q does not exist (resolved to %s)", filePart, targetPath)})
						}
						continue
					}
				}

				if !hasFrag || fragPart == "" {
					continue
				}

				info := anchors[targetPath]
				if info.ids[fragPart] || info.slugs[fragPart] {
					continue
				}
				findings = append(findings, finding{f.path, lineNo,
					fmt.Sprintf("link fragment %q does not resolve in %s", fragPart, targetPath)})
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
// not an oversight. No fenced block in this corpus currently contains a "### Consequence" line or
// a quotation span, so Checks C and D have no live exposure to this asymmetry either way.

var (
	decisionFileRe = regexp.MustCompile(`^decision-([DT])(\d+)\.md$`)
	citeDRe        = regexp.MustCompile(`\bD(\d+)\b`)
	citeTRe        = regexp.MustCompile(`\bT(\d+)\b`)
	citeFRe        = regexp.MustCompile(`\bF(\d+)\b`)

	// uatFindingRe is the UAT finding ID shape (`U4.F1`). Backlog tasks carry those, and the
	// zero-F-refs rule exists to catch a citation of a log that never existed, not these.
	uatFindingRe = regexp.MustCompile(`\bU\d+\.F\d+\b`)
)

// decisionFiles returns every decision-{D,T}<n>.md file in byPath, grouped by prefix ("D"/"T") and
// number. Files in backlog/decisions that do not match the name shape are reported by Check C.
func decisionFiles(byPath map[string]*docFile) map[string]map[int][]*docFile {
	out := map[string]map[int][]*docFile{"D": {}, "T": {}}
	for _, f := range byPath {
		if f.dir() != decisionsDir {
			continue
		}
		m := decisionFileRe.FindStringSubmatch(f.base)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		out[m[1]][n] = append(out[m[1]][n], f)
	}
	return out
}

func checkCitations(files []*docFile, byPath map[string]*docFile) []finding {
	var findings []finding
	valid := decisionFiles(byPath)

	for _, f := range files {
		for lineIdx, line := range f.lines {
			lineNo := lineIdx + 1

			for _, m := range citeDRe.FindAllStringSubmatch(line, -1) {
				n, _ := strconv.Atoi(m[1])
				if len(valid["D"][n]) == 0 {
					findings = append(findings, finding{f.path, lineNo,
						fmt.Sprintf("citation D%d does not resolve to %s/decision-D%d.md", n, decisionsDir, n)})
				}
			}
			for _, m := range citeTRe.FindAllStringSubmatch(line, -1) {
				n, _ := strconv.Atoi(m[1])
				if len(valid["T"][n]) == 0 {
					findings = append(findings, finding{f.path, lineNo,
						fmt.Sprintf("citation T%d does not resolve to %s/decision-T%d.md", n, decisionsDir, n)})
				}
			}
			for _, m := range citeFRe.FindAllStringSubmatch(uatFindingRe.ReplaceAllString(line, ""), -1) {
				findings = append(findings, finding{f.path, lineNo,
					fmt.Sprintf("F-log reference %q found; no F-log has ever existed in this project", m[0])})
			}
		}
	}
	return findings
}

// ---------------------------------------------------------------------------------------------
// Check C: per-namespace decision invariants (backlog/decisions, D and T)
// ---------------------------------------------------------------------------------------------

var (
	frontMatterKeyRe = regexp.MustCompile(`^(id|title|date|status):\s*(.*)$`)
	isoDateRe        = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
	consequenceRe    = regexp.MustCompile(`^### Consequence`)
)

func checkLogInvariants(byPath map[string]*docFile) []finding {
	var findings []finding

	// Every file in backlog/decisions must be a hand-authored decision-D<n>/T<n> file: anything
	// else is what `backlog decision create` mints (T56) — a third ID namespace.
	var paths []string
	for p, f := range byPath {
		if f.dir() == decisionsDir {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	for _, p := range paths {
		f := byPath[p]
		m := decisionFileRe.FindStringSubmatch(f.base)
		if m == nil {
			findings = append(findings, finding{f.path, 0,
				"decision file name must be decision-D<n>.md or decision-T<n>.md (hand-authored; never `backlog decision create`, T56)"})
			continue
		}
		if n, _ := strconv.Atoi(m[2]); strconv.Itoa(n) != m[2] {
			findings = append(findings, finding{f.path, 0, "decision number must not be zero-padded"})
		}
		findings = append(findings, checkEntryFile(f, m[1], m[2])...)
	}

	// IDs contiguous from 1, per namespace, no duplicates.
	grouped := decisionFiles(byPath)
	for _, prefix := range []string{"D", "T"} {
		maxN := 0
		for n, fs := range grouped[prefix] {
			if n > maxN {
				maxN = n
			}
			if len(fs) > 1 {
				findings = append(findings, finding{fs[0].path, 0,
					fmt.Sprintf("%s%d appears in %d files (duplicate)", prefix, n, len(fs))})
			}
		}
		// Walk the numbers that exist, not 1..maxN: a stray decision-D999999999.md must not make
		// this loop (or the report) enormous. Report the first few missing IDs and a remainder.
		var nums []int
		for n := range grouped[prefix] {
			nums = append(nums, n)
		}
		sort.Ints(nums)
		const maxListed = 5
		var listed []string
		missing, prev := 0, 0
		for _, n := range nums {
			for g := prev + 1; g < n && len(listed) < maxListed; g++ {
				listed = append(listed, fmt.Sprintf("%s%d", prefix, g))
			}
			if n > prev+1 {
				missing += n - prev - 1
			}
			if n > prev {
				prev = n
			}
		}
		if missing > 0 {
			verb := "is"
			if len(listed) > 1 {
				verb = "are"
			}
			msg := fmt.Sprintf("%s %s missing (gap in %s1..%s%d)", strings.Join(listed, ", "), verb, prefix, prefix, maxN)
			if missing > len(listed) {
				msg = fmt.Sprintf("%s and %d more are missing (gap in %s1..%s%d)", strings.Join(listed, ", "), missing-len(listed), prefix, prefix, maxN)
			}
			findings = append(findings, finding{decisionsDir, 0, msg})
		}
	}
	return findings
}

// unquote strips one layer of matching single or double quotes from a front matter value.
func unquote(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && (v[0] == '\'' && v[len(v)-1] == '\'' || v[0] == '"' && v[len(v)-1] == '"') {
		return v[1 : len(v)-1]
	}
	return v
}

// checkEntryFile verifies one decision file: front matter whose id agrees with the filename and
// that carries a title, an ISO date, and a status; and a body with a line starting
// "### Consequence" (a prefix match, so the "### Consequence if accepted" variant counts too).
func checkEntryFile(f *docFile, prefix, num string) []finding {
	var findings []finding
	wantID := "decision-" + prefix + num

	if len(f.lines) == 0 || f.lines[0] != "---" {
		return []finding{{f.path, 1, "decision file has no YAML front matter"}}
	}
	end := -1
	for i := 1; i < len(f.lines); i++ {
		if f.lines[i] == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return []finding{{f.path, 1, "decision file front matter is never closed"}}
	}

	got := map[string]string{}
	for _, line := range f.lines[1:end] {
		if m := frontMatterKeyRe.FindStringSubmatch(line); m != nil {
			got[m[1]] = unquote(m[2])
		}
	}
	if got["id"] != wantID {
		findings = append(findings, finding{f.path, 1,
			fmt.Sprintf("front matter id %q does not match filename (want %q)", got["id"], wantID)})
	}
	if got["title"] == "" {
		findings = append(findings, finding{f.path, 1, "front matter has no title"})
	}
	if !isoDateRe.MatchString(got["date"]) {
		findings = append(findings, finding{f.path, 1,
			fmt.Sprintf("front matter date %q is not an ISO date", got["date"])})
	}
	if got["status"] == "" {
		findings = append(findings, finding{f.path, 1, "front matter has no status"})
	}

	hasConsequence := false
	for _, line := range f.lines[end+1:] {
		if consequenceRe.MatchString(line) {
			hasConsequence = true
			break
		}
	}
	if !hasConsequence {
		findings = append(findings, finding{f.path, end + 2,
			fmt.Sprintf("%s%s has no line starting with \"### Consequence\"", prefix, num)})
	}
	return findings
}

// ---------------------------------------------------------------------------------------------
// Check D: verbatim quotation checking
// ---------------------------------------------------------------------------------------------

var (
	// project-assessment-2026-08.md is included alongside the other cross-referenced docs:
	// T27's entry links and quotes it directly (docs/project-assessment-2026-08.md §6's Phase 1/2
	// text), so it is a real, checkable citation target in this corpus too.
	//
	// Links are written relative to the citing file ("../../docs/design.md" from a decision file,
	// "../backlog/decisions/decision-D9.md" from docs/), so the regexp accepts any directory prefix
	// and captures only the base name. A link to a single decision file is attributed to its whole
	// namespace (see buildSources), exactly as a bare Dn/Tn token is — the pre-T56 behavior, when
	// the namespace was one file.
	citeLinkRe   = regexp.MustCompile(`\]\((?:[^)\s#]*/)?(design\.md|tdd\.md|roadmap\.md|decision-log\.md|tech-decision-log\.md|project-assessment-2026-08\.md|decision-[DT]\d+\.md)(?:#[^)]*)?\)`)
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

// sourceKey maps a cited file's base name to the key buildSources indexes it under: a decision
// file collapses to its namespace's log name.
func sourceKey(base string) string {
	if m := decisionFileRe.FindStringSubmatch(base); m != nil {
		if m[1] == "D" {
			return "decision-log.md"
		}
		return "tech-decision-log.md"
	}
	return base
}

// buildSources returns the checkable text behind each citation source key. The two log keys are
// the concatenation of the docs/ stub (which still carries the log's rules) and every decision
// file of that namespace, so a quotation of a log rule or of any entry verifies the same way it
// did when the log was one file.
func buildSources(byPath map[string]*docFile) map[string]string {
	src := map[string]string{}
	for _, name := range []string{"design.md", "tdd.md", "roadmap.md", "project-assessment-2026-08.md"} {
		if f, ok := byPath[docsDir+"/"+name]; ok {
			src[name] = f.content
		}
	}
	for prefix, key := range map[string]string{"D": "decision-log.md", "T": "tech-decision-log.md"} {
		var sb strings.Builder
		if f, ok := byPath[docsDir+"/"+key]; ok {
			sb.WriteString(f.content)
		}
		groups := decisionFiles(byPath)[prefix]
		var nums []int
		for n := range groups {
			nums = append(nums, n)
		}
		sort.Ints(nums)
		for _, n := range nums {
			for _, f := range groups[n] {
				sb.WriteString("\n")
				sb.WriteString(f.content)
			}
		}
		src[key] = sb.String()
	}
	return src
}

// findCitationSource looks for the citation patterns (a markdown link to a tracked doc or decision
// file, a bare backtick-quoted filename, or a bare Dn/Tn/Pn/Jn token) within before, and returns
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
		consider(m[0], m[1], sourceKey(before[m[2]:m[3]]))
	}
	for _, m := range citeBacktick.FindAllStringSubmatchIndex(before, -1) {
		consider(m[0], m[1], sourceKey(before[m[2]:m[3]]))
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
		file:   "decision-T7.md",
		quoted: "Only as a last resort, a hasp settings file — and no case has yet required one.",
	},
	{
		// decision-log.md D16 itself (dated 2026-08-29) is the entry that amends P9's final
		// clause, and quotes P9 as it read the moment before that amendment took effect — the very
		// same historical clause T7 above preserves, from the other side of the same event. D16's
		// own next sentence names the amendment explicitly ("docs/design.md §4 states the amended
		// form"), so this is not a checker-missed drift: design.md's current P9 (§4) no longer
		// carries this clause on purpose.
		file:   "decision-D16.md",
		quoted: "No case has yet required one",
	},
	{
		// tech-decision-log.md T27 (dated 2026-08-29) quotes design.md §10's "implementation
		// stack" deferral item as it read before this same entry closed it — T27's own sentence
		// says so directly: "Before this entry closed it, the item read: ...". design.md §10 no
		// longer carries a deferred implementation-stack item because T27 is exactly the decision
		// that resolved it (Go, per docs/tdd.md §2), so its absence from the current doc is the
		// intended outcome T27 records, not drift.
		file:   "decision-T27.md",
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

func checkQuotations(files []*docFile, sources map[string]string) []finding {
	var findings []finding
	for _, f := range files {
		if !f.inQuoteScope() {
			continue
		}
		for _, para := range splitParagraphs(f) {
			for _, q := range findQuotes(para.text) {
				before := sentenceWindow(para.text[:q.start])
				source, ok := findCitationSource(before)
				if !ok {
					continue // unattributed/external quote: not this tool's to check
				}
				srcContent, haveSource := sources[source]
				if !haveSource {
					continue // the named source file doesn't exist; Check A covers that
				}
				if wordNormalize(q.quoted) == "" || quoteMatchesSource(q.quoted, srcContent) {
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
