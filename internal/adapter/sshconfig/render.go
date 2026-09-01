package sshconfig

// Render reconstructs f's bytes. Outside a MarkedRegion, this is byte-exact for arbitrary input
// (Render(Parse(b)) == b, T2, fuzz-tested); inside one, D7 elaboration 1 licenses hasp to
// regenerate freely — out of scope for M1, which never writes.
func (f *File) Render() []byte {
	var out []byte
	for _, n := range f.Nodes {
		out = append(out, renderNode(n)...)
	}
	return out
}

// RenderNodes renders an arbitrary node slice to bytes, without exposing renderNode itself. This
// is how an app-layer use case (e.g. `new host`, M3) renders a freshly-built MarkedRegion or
// HostBlock it constructed itself, before either splicing it into a WriteRegion Change's After or
// (for a whole-file-owned group file, D7) using it as the entire file's new content.
func RenderNodes(nodes []Node) []byte { return (&File{Nodes: nodes}).Render() }

// appendNodeWithSeparator appends next onto out, inserting a single '\n' separator first if out
// is non-empty, next is non-empty, and out's last byte is not already a newline (M3 close-out
// review finding 2, shared between HostBlock.render() and MarkedRegion.render()'s body loop, T2/
// T17/D7).
//
// This is a true no-op for anything Parse itself ever produces: only the input's absolute last
// physical line can lack a terminator, and by construction that can only be the final node
// appended in either caller's loop — nothing follows it there to glue onto. It stops being a
// no-op once app-layer code builds a node stream itself from an untouched, possibly-terminator-
// less node (not necessarily the stream's last element) — e.g. edit host's
// applyDirectiveFieldEdits appending a freshly-authored Directive after a preserved one that
// happens to lack a trailing newline (HostBlock.render()'s original bug), or adopt host's
// newManagedRegion wrapping an already-parsed, file-final HostBlock into a MarkedRegion's Body
// (MarkedRegion.render()'s Begin/Body/End junction bugs). Without this guard, two adjacent
// rendered parts glue into one physical line and either corrupt a preserved directive's value or
// make scanMarkers stop recognizing hasp's own End marker as its own line on the next Parse
// (DefectUnmatchedBegin) — the same bug class both callers hit independently.
//
// Per MarkedRegion's own doc comment, Body's contract inside a region is already the weaker
// Render(Parse(Render(r))) == Render(r) (idempotent), not byte-exact for arbitrary content — D7
// elaboration 1 licenses hasp to regenerate a region's body wholesale — so inserting this one
// separator byte (never rewriting any byte a preserved node already produced) does not touch
// T2/P2's stricter byte-exact guarantee, which is scoped to outside a marked region.
func appendNodeWithSeparator(out []byte, next []byte) []byte {
	if len(next) > 0 && len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	return append(out, next...)
}

// renderNode dispatches on Node's concrete type. A type switch here, rather than an exported
// method on the Node interface, keeps Node itself the minimal marker interface tdd.md §3
// sketches while centralizing every node kind's byte-reconstruction logic in one place.
func renderNode(n Node) []byte {
	switch v := n.(type) {
	case *Line:
		return v.render()
	case *Directive:
		return v.render()
	case *HostBlock:
		return v.render()
	case *MarkedRegion:
		return v.render()
	case *MetadataLine:
		return v.render()
	default:
		return nil
	}
}
