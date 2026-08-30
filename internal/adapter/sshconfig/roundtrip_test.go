package sshconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func assertRoundTrip(t *testing.T, b []byte) {
	t.Helper()
	got := Parse(b).Render()
	if !bytes.Equal(got, b) {
		t.Errorf("round-trip mismatch:\n got: %q\nwant: %q", got, b)
	}
}

func TestRoundTrip_Separators(t *testing.T) {
	cases := []string{
		"Port 22\n",
		"Port = 22\n",
		"Port=22\n",
		"Port =22\n",
		"Port= 22\n",
		"    Port 22\n", // indented, as inside a Host block
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) { assertRoundTrip(t, []byte(c)) })
	}
}

func TestRoundTrip_QuotedHashNotAComment(t *testing.T) {
	assertRoundTrip(t, []byte(`ProxyCommand nc -x proxy 1080 %h %p "arg # not a comment"`+"\n"))
}

func TestRoundTrip_TrailingComment(t *testing.T) {
	assertRoundTrip(t, []byte("Port 22 # the default port\n"))
	assertRoundTrip(t, []byte("Port 22# no space before hash\n"))
}

func TestRoundTrip_CommentAndBlankLines(t *testing.T) {
	assertRoundTrip(t, []byte("# a comment\n\nPort 22\n"))
}

func TestRoundTrip_UnrecognizedKeyword(t *testing.T) {
	assertRoundTrip(t, []byte("SomeFutureDirective value here\n"))
}

func TestRoundTrip_NoTrailingNewline(t *testing.T) {
	assertRoundTrip(t, []byte("Port 22"))
	assertRoundTrip(t, []byte("HostName example.com"))
}

func TestRoundTrip_Empty(t *testing.T) {
	assertRoundTrip(t, []byte(""))
}

func TestRoundTrip_CRLF(t *testing.T) {
	assertRoundTrip(t, []byte("Port 22\r\nHostName example.com\r\n"))
}

func TestRoundTrip_MixedLineEndings(t *testing.T) {
	assertRoundTrip(t, []byte("Port 22\r\nHostName example.com\nUser jesse\r\n"))
}

func TestDirective_ArgsQuoteAware(t *testing.T) {
	f := Parse([]byte(`IdentityFile "~/my key"` + "\n"))
	d, ok := f.Nodes[0].(*Directive)
	if !ok {
		t.Fatalf("Nodes[0] = %T, want *Directive", f.Nodes[0])
	}
	args := d.Args()
	if len(args) != 1 || args[0] != "~/my key" {
		t.Errorf("Args() = %v, want [\"~/my key\"]", args)
	}
}

func TestDirective_ArgsMultiple(t *testing.T) {
	f := Parse([]byte("IdentityFile ~/.ssh/a ~/.ssh/b\n"))
	d := f.Nodes[0].(*Directive)
	args := d.Args()
	if len(args) != 2 || args[0] != "~/.ssh/a" || args[1] != "~/.ssh/b" {
		t.Errorf("Args() = %v", args)
	}
}

func TestDirective_KeywordCasePreserved(t *testing.T) {
	f := Parse([]byte("hostname example.com\n"))
	d, ok := f.Nodes[0].(*Directive)
	if !ok {
		t.Fatalf("Nodes[0] = %T, want *Directive (lowercase keyword still recognized)", f.Nodes[0])
	}
	if d.Keyword != "hostname" {
		t.Errorf("Keyword = %q, want %q (case preserved as written)", d.Keyword, "hostname")
	}
}

func TestClassifyLine_UnrecognizedKeywordIsOpaqueLine(t *testing.T) {
	f := Parse([]byte("TotallyMadeUpDirective foo\n"))
	if _, ok := f.Nodes[0].(*Line); !ok {
		t.Fatalf("Nodes[0] = %T, want *Line (unrecognized keyword stays opaque)", f.Nodes[0])
	}
}

// TestRoundTrip_ExistingFixtures re-runs the three M0 CST shape fixtures (crlf, empty,
// no-trailing-newline) through this package directly, on top of fixtures_test.go's own
// M0-era proof that they exist and have the expected byte shape.
func TestRoundTrip_ExistingFixtures(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "testdata", "sshconfig")

	for _, name := range []string{"crlf", "empty", "no-trailing-newline"} {
		t.Run(name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			assertRoundTrip(t, b)
		})
	}
}
