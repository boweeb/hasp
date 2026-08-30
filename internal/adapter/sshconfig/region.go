package sshconfig

import (
	"bytes"
	"strings"
)

// The sentinel lines hasp writes to delimit its managed region inside a co-owned file (D7's
// harder case, tdd.md §6). Matched by exact byte equality against a physical line's content
// (no leading/trailing whitespace tolerance) — these are "the exact marker lines hasp wrote,"
// and a line that doesn't match exactly is simply not recognized as a marker, not a defect.
const (
	markedRegionBeginSentinel = "# >>> hasp:managed >>>"
	markedRegionEndSentinel   = "# <<< hasp:managed <<<"
)

// metadataSentinel prefixes a #:hasp key = value line (§7, T25) inside a MarkedRegion body.
const metadataSentinel = "#:hasp"

// MarkedRegion is a delimited span inside a configuration file that hasp owns outright (D7).
// Everything inside Body is hasp's own; on a write it would be regenerated wholesale (D7
// elaboration 1) — out of scope for M1, which never writes. Because of that, Body's contract
// is only idempotent (Render(Parse(Render(r))) == Render(r) for a body r hasp produced), not
// byte-exact for arbitrary human-edited content — the weaker half of T2's contract.
type MarkedRegion struct {
	Begin, End []byte // the exact marker lines hasp wrote, verbatim including their terminator
	Body       []Node // hasp-owned; may contain MetadataLine alongside Directive/HostBlock/Line
}

func (*MarkedRegion) node() {}

func (r *MarkedRegion) render() []byte {
	if r == nil {
		return nil
	}
	out := append([]byte{}, r.Begin...)
	for _, n := range r.Body {
		out = append(out, renderNode(n)...)
	}
	out = append(out, r.End...)
	return out
}

// MetadataLine is a "#:hasp key = value" line inside a MarkedRegion body — its own node type
// rather than a re-scanned Line (T25). Key and Value are parsed, not raw-byte spans: unlike
// Directive, a metadata line's exact human spelling is never preserved on Render, because D7
// elaboration 1 already licenses hasp to regenerate a region's body wholesale — canonical
// re-formatting on Render is exactly that license exercised, not a P2 violation, since P2's
// byte-exact guarantee is scoped to outside a marked region (T2).
type MetadataLine struct {
	Key, Value string
	Terminator []byte
}

func (*MetadataLine) node() {}

func (m *MetadataLine) render() []byte {
	if m == nil {
		return nil
	}
	var out []byte
	out = append(out, metadataSentinel...)
	out = append(out, ' ')
	out = append(out, m.Key...)
	out = append(out, " = "...)
	out = append(out, m.Value...)
	out = append(out, m.Terminator...)
	return out
}

// tryMetadataLine recognizes "#:hasp <key> = <value>" syntax. Sentinel presence, not comment
// syntax, is what marks a line as metadata (§7, mirroring D15's presence-not-contents rule for
// the .hasp marker) — an ordinary "#" comment hasp itself writes inside a region is never
// mistaken for this.
func tryMetadataLine(pl physicalLine) (*MetadataLine, bool) {
	indentEnd := 0
	for indentEnd < len(pl.Content) && isSpace(pl.Content[indentEnd]) {
		indentEnd++
	}
	content := pl.Content[indentEnd:]
	if !bytes.HasPrefix(content, []byte(metadataSentinel)) {
		return nil, false
	}
	rest := content[len(metadataSentinel):]
	if len(rest) == 0 || !isSpace(rest[0]) {
		return nil, false // "#:haspfoo" is not the sentinel
	}
	rest = bytes.TrimLeft(rest, " \t")

	eqIdx := bytes.IndexByte(rest, '=')
	if eqIdx == -1 {
		return nil, false
	}
	key := strings.TrimSpace(string(rest[:eqIdx]))
	if key == "" {
		return nil, false
	}
	value := strings.TrimSpace(string(rest[eqIdx+1:]))

	return &MetadataLine{Key: key, Value: value, Terminator: pl.Terminator}, true
}

// MarkerDefectKind enumerates the detectable ways a marker pair can be malformed (T18).
type MarkerDefectKind int

const (
	DefectUnmatchedBegin    MarkerDefectKind = iota // a begin marker with no matching end
	DefectUnmatchedEnd                              // an end marker with no matching begin
	DefectDuplicateBegin                            // a second begin marker before the first is closed
	DefectEndBeforeBegin                            // an end marker with no begin preceding it
	DefectNestedInHostBlock                         // a marker line found inside a Host/Match block
)

func (k MarkerDefectKind) String() string {
	switch k {
	case DefectUnmatchedBegin:
		return "unmatched-begin"
	case DefectUnmatchedEnd:
		return "unmatched-end"
	case DefectDuplicateBegin:
		return "duplicate-begin"
	case DefectEndBeforeBegin:
		return "end-before-begin"
	case DefectNestedInHostBlock:
		return "nested-in-host-block"
	default:
		return "unknown"
	}
}

// MarkerDefect is one detected way a marker pair is malformed, attached to File as data rather
// than returned as a Parse error (T18): a damaged marker means hasp cannot know where its own
// territory ends, so writing must fail closed — but reading must not, or J1's "surveying any
// machine works" promise breaks on exactly the files most in need of being left alone.
type MarkerDefect struct {
	Kind   MarkerDefectKind
	Line   int // 1-based line number in the input, for a human-facing report
	Detail string
}

// scanMarkers finds hasp's marker sentinel pair, if a well-formed one exists, and records every
// detectable defect. It supports carving out at most one MarkedRegion per file — hasp only ever
// writes one — and returns beginIdx == -1 (or endIdx == -1) when there is none to carve, with
// every anomaly recorded in defects.
//
// A marker line found after the file's first Host/Match line is "nested in a Host/Match block":
// ssh_config has no way to return to top-level scope within one parse pass once a Host/Match
// line has appeared (every later line belongs to some block), so this is a simple latching flag,
// not a depth counter.
func scanMarkers(lines []physicalLine) (beginIdx, endIdx int, defects []MarkerDefect) {
	beginIdx, endIdx = -1, -1

	const (
		stateNone = iota
		stateOpen
		stateClosed
	)
	state := stateNone
	inHostBlock := false

	for i, pl := range lines {
		if kw, _, _, ok := lineKeyword(pl.Content); ok && isHostOrMatchKeyword(kw) {
			inHostBlock = true
		}

		switch string(pl.Content) {
		case markedRegionBeginSentinel:
			if inHostBlock {
				defects = append(defects, MarkerDefect{Kind: DefectNestedInHostBlock, Line: i + 1, Detail: "begin marker found inside a Host/Match block"})
				continue
			}
			switch state {
			case stateNone:
				state = stateOpen
				beginIdx = i
			case stateOpen:
				defects = append(defects, MarkerDefect{Kind: DefectDuplicateBegin, Line: i + 1, Detail: "a second begin marker appeared before the first was closed"})
			case stateClosed:
				defects = append(defects, MarkerDefect{Kind: DefectDuplicateBegin, Line: i + 1, Detail: "a second begin marker appeared after an earlier region already closed"})
			}
		case markedRegionEndSentinel:
			if inHostBlock {
				defects = append(defects, MarkerDefect{Kind: DefectNestedInHostBlock, Line: i + 1, Detail: "end marker found inside a Host/Match block"})
				continue
			}
			switch state {
			case stateNone:
				defects = append(defects, MarkerDefect{Kind: DefectEndBeforeBegin, Line: i + 1, Detail: "end marker with no begin preceding it"})
			case stateOpen:
				state = stateClosed
				endIdx = i
			case stateClosed:
				defects = append(defects, MarkerDefect{Kind: DefectUnmatchedEnd, Line: i + 1, Detail: "end marker with no open begin to match"})
			}
		}
	}

	if state == stateOpen {
		defects = append(defects, MarkerDefect{Kind: DefectUnmatchedBegin, Line: beginIdx + 1, Detail: "begin marker with no matching end"})
	}

	return beginIdx, endIdx, defects
}
