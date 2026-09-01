package sshconfig

import "strings"

// HostBlock groups the directives following a "Host" or "Match" line until the next block
// boundary. Match blocks are recognized structurally, here, so trailing directives are never
// mis-grouped into the preceding Host block — but produce no domain.Host in M1 (app layer
// consumes only blocks whose Header.Keyword is "Host", case-insensitively).
type HostBlock struct {
	// Header is the "Host ..." or "Match ..." line itself, reusing Directive's raw-byte-
	// preserving shape (LeadingTrivia/Keyword/RawValue/Trivia/Terminator) rather than tdd.md §3's
	// literal sketch, which has no field for the block header line's own raw bytes. This is a
	// deliberate, documented departure — without it, Render could not reconstruct the header
	// line byte-exactly.
	Header   *Directive
	Patterns []string // Header.Args(), cached at parse time for convenient access
	// Directives holds every node between this block's Header and the next block boundary (or
	// end of input): *Directive for recognized keywords, *Line for everything else — never an
	// error, matching tdd.md §3's sketch exactly for this field.
	Directives []Node
}

func (*HostBlock) node() {}

func (h *HostBlock) render() []byte {
	if h == nil {
		return nil
	}
	out := h.Header.render()
	// Found during M3's close-out audit (the same bug class as region.go's MarkedRegion.render()
	// fix, one layer down; consolidated into the shared appendNodeWithSeparator helper alongside
	// it, T2/T17/D7): a HostBlock's own Header/Directives are joined here, one node at a time, so a
	// freshly-authored Directive appended after an untouched, terminator-less preserved one (e.g.
	// edit host's applyDirectiveFieldEdits, or a custom host-group file's last stanza) never glues
	// onto the preserved node's value.
	for _, n := range h.Directives {
		out = appendNodeWithSeparator(out, renderNode(n))
	}
	return out
}

// isBlockHeader reports whether n is a Host or Match block header, as classifyLine produces it.
func isBlockHeader(n Node) (*Directive, bool) {
	d, ok := n.(*Directive)
	if !ok {
		return nil, false
	}
	lower := strings.ToLower(d.Keyword)
	if lower != "host" && lower != "match" {
		return nil, false
	}
	return d, true
}

// groupHostBlocks re-scans a flat node stream, grouping each Host/Match header and the nodes
// that follow it (up to the next header or end of input) into a HostBlock. ssh_config has no
// nesting — a new Host/Match line always starts a sibling block, never a child — so this is a
// single linear pass with no recursion.
func groupHostBlocks(nodes []Node) []Node {
	var out []Node
	var current *HostBlock

	flush := func() {
		if current != nil {
			out = append(out, current)
			current = nil
		}
	}

	for _, n := range nodes {
		if header, ok := isBlockHeader(n); ok {
			flush()
			current = &HostBlock{Header: header, Patterns: header.Args()}
			continue
		}
		if current != nil {
			current.Directives = append(current.Directives, n)
		} else {
			out = append(out, n)
		}
	}
	flush()

	return out
}
