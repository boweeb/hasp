package sshconfig

import "testing"

func TestRoundTrip_HostBlock(t *testing.T) {
	assertRoundTrip(t, []byte("Host work\n    HostName work.example.com\n    User jesse\n    IdentityFile ~/.ssh/w\n"))
}

func TestRoundTrip_MultipleHostBlocks(t *testing.T) {
	assertRoundTrip(t, []byte("Host work\n    HostName work.example.com\n\nHost personal\n    HostName home.example.com\n"))
}

func TestRoundTrip_MatchBlockBoundary(t *testing.T) {
	assertRoundTrip(t, []byte("Host work\n    HostName work.example.com\n\nMatch host personal\n    User jesse\n"))
}

func TestRoundTrip_TrailingContentAfterHostBlock(t *testing.T) {
	assertRoundTrip(t, []byte("Host work\n    HostName work.example.com\n\n# trailing comment, not inside any block\n"))
}

func TestParse_HostBlockPatterns(t *testing.T) {
	f := Parse([]byte("Host work personal\n    HostName example.com\n"))
	if len(f.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(f.Nodes))
	}
	hb, ok := f.Nodes[0].(*HostBlock)
	if !ok {
		t.Fatalf("Nodes[0] = %T, want *HostBlock", f.Nodes[0])
	}
	if len(hb.Patterns) != 2 || hb.Patterns[0] != "work" || hb.Patterns[1] != "personal" {
		t.Errorf("Patterns = %v, want [work personal]", hb.Patterns)
	}
	if len(hb.Directives) != 1 {
		t.Fatalf("got %d directives in block, want 1", len(hb.Directives))
	}
	d, ok := hb.Directives[0].(*Directive)
	if !ok || d.Keyword != "HostName" {
		t.Errorf("Directives[0] = %#v, want a HostName Directive", hb.Directives[0])
	}
}

func TestParse_MatchBlockNotConsumedAsHost(t *testing.T) {
	f := Parse([]byte("Host work\n    HostName a\n\nMatch host personal\n    User jesse\n"))
	if len(f.Nodes) != 2 {
		t.Fatalf("got %d top-level nodes, want 2 (one HostBlock per header)", len(f.Nodes))
	}
	first, ok := f.Nodes[0].(*HostBlock)
	if !ok || first.Header.Keyword != "Host" {
		t.Fatalf("Nodes[0] = %#v, want a Host block", f.Nodes[0])
	}
	second, ok := f.Nodes[1].(*HostBlock)
	if !ok || second.Header.Keyword != "Match" {
		t.Fatalf("Nodes[1] = %#v, want a Match block", f.Nodes[1])
	}
}

func TestParse_UnrecognizedDirectiveInsideHostBlockStaysOpaque(t *testing.T) {
	f := Parse([]byte("Host work\n    SomeFutureDirective foo\n"))
	hb := f.Nodes[0].(*HostBlock)
	if _, ok := hb.Directives[0].(*Line); !ok {
		t.Errorf("Directives[0] = %T, want *Line (unrecognized keyword stays opaque even inside a block)", hb.Directives[0])
	}
}
