package sshconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// FuzzSSHConfigRoundTrip is T2's contract, tested directly: Render(Parse(b)) == b for arbitrary
// bytes outside a marked region; Render(Parse(Render(r))) == Render(r) — idempotent — for a
// region body, since D7 elaboration 1 licenses hasp to reformat what it owns. The contract is
// stated at exactly this two-tier granularity in tdd.md §6 ("the contract is weaker by design
// [inside a region]... but still idempotent"); a single unconditional byte-equality assertion
// would be a stricter test than the ratified contract actually requires, and would fail
// spuriously the moment a fuzzed mutation lands a non-canonically-spaced #:hasp metadata line
// inside an otherwise well-formed region (MetadataLine.Render canonicalizes, per T25 — see
// region.go).
func FuzzSSHConfigRoundTrip(f *testing.F) {
	for _, seed := range fuzzSeedCorpus(f) {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, b []byte) {
		parsed := Parse(b)
		got := parsed.Render()

		if !hasMarkedRegion(parsed.Nodes) {
			if !bytes.Equal(got, b) {
				t.Fatalf("round-trip mismatch outside any marked region:\n got: %q\nwant: %q", got, b)
			}
			return
		}

		again := Parse(got).Render()
		if !bytes.Equal(again, got) {
			t.Fatalf("region render is not idempotent:\n first: %q\nsecond: %q", got, again)
		}
	})
}

func hasMarkedRegion(nodes []Node) bool {
	for _, n := range nodes {
		if _, ok := n.(*MarkedRegion); ok {
			return true
		}
	}
	return false
}

// fuzzSeedCorpus loads the committed CST shape fixtures under testdata/sshconfig/ (crlf, empty,
// no-trailing-newline — T24) plus inline snippets covering every node kind this package
// understands: Host/Match blocks, a well-formed region with metadata, one seed per
// MarkerDefectKind, mixed line endings, and a quoted '#' that is not a comment.
func fuzzSeedCorpus(f *testing.F) [][]byte {
	f.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		f.Fatal("runtime.Caller failed")
	}
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "testdata", "sshconfig")

	entries, err := os.ReadDir(dir)
	if err != nil {
		f.Fatalf("read %s: %v", dir, err)
	}

	var seeds [][]byte
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			f.Fatalf("read %s: %v", e.Name(), err)
		}
		seeds = append(seeds, b)
	}

	seeds = append(seeds,
		[]byte("Host work\n    HostName work.example.com\n    User jesse\n    IdentityFile ~/.ssh/w\n"),
		[]byte(wellFormedRegion),
		[]byte("# >>> hasp:managed >>>\n#:hasp created_by = \"new host\"\nInclude ~/.ssh/work.sshconfig\n# <<< hasp:managed <<<\n"),
		[]byte("# >>> hasp:managed >>>\nInclude ~/.ssh/work.sshconfig\n"), // unmatched begin
		[]byte("# <<< hasp:managed <<<\n"), // end before begin
		[]byte("# >>> hasp:managed >>>\n# >>> hasp:managed >>>\nInclude ~/.ssh/work.sshconfig\n# <<< hasp:managed <<<\n"),              // duplicate begin
		[]byte("# >>> hasp:managed >>>\nInclude ~/.ssh/work.sshconfig\n# <<< hasp:managed <<<\n# <<< hasp:managed <<<\n"),              // unmatched end
		[]byte("Host work\n    HostName example.com\n# >>> hasp:managed >>>\nInclude ~/.ssh/work.sshconfig\n# <<< hasp:managed <<<\n"), // nested in host block
		[]byte("Port 22\r\nHostName example.com\r\n"),
		[]byte(`ProxyCommand nc -x proxy 1080 %h %p "arg # not a comment"`+"\n"),
	)

	return seeds
}
