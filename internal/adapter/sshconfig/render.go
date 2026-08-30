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
