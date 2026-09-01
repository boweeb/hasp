package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/boweeb/hasp/internal/app"
)

func TestPassphraseFlags_Explicit_MutuallyExclusive(t *testing.T) {
	f := passphraseFlags{Passphrase: true, NoPassphrase: true}
	if _, _, err := f.explicit(); !errors.Is(err, app.ErrUsage) {
		t.Errorf("explicit() with two flags set: err = %v, want ErrUsage", err)
	}
}

func TestPassphraseFlags_Explicit_NoneSet(t *testing.T) {
	_, ok, err := (passphraseFlags{}).explicit()
	if err != nil {
		t.Fatalf("explicit(): %v", err)
	}
	if ok {
		t.Error("explicit() with no flags set: ok = true, want false")
	}
}

func TestPassphraseFlags_Explicit_EachFlag(t *testing.T) {
	cases := []struct {
		flags passphraseFlags
		want  passphraseMode
	}{
		{passphraseFlags{Passphrase: true}, passphrasePrompt},
		{passphraseFlags{NoPassphrase: true}, passphraseNone},
		{passphraseFlags{PassphraseStdin: true}, passphraseStdin},
	}
	for _, c := range cases {
		mode, ok, err := c.flags.explicit()
		if err != nil {
			t.Errorf("explicit(%+v): %v", c.flags, err)
			continue
		}
		if !ok || mode != c.want {
			t.Errorf("explicit(%+v) = (%v, %v), want (%v, true)", c.flags, mode, ok, c.want)
		}
	}
}

func noEnv(string) (string, bool) { return "", false }

// TestResolvePassphraseMode_Precedence is the explicit precedence test the task calls for: flag
// beats env beats settings beats the built-in default, with all four sources contributing at
// once, exercised as four progressively-narrowing cases (tdd.md §9, T6).
func TestResolvePassphraseMode_Precedence(t *testing.T) {
	env := func(string) (string, bool) { return "stdin", true }

	// All four contribute: flag wins.
	mode, err := resolvePassphraseMode(passphraseFlags{NoPassphrase: true}, env, "prompt", true)
	if err != nil {
		t.Fatalf("resolvePassphraseMode: %v", err)
	}
	if mode != passphraseNone {
		t.Errorf("flag+env+settings+default all present: mode = %v, want %v (flag wins)", mode, passphraseNone)
	}

	// No flag: env, settings, and default all contribute; env wins.
	mode, err = resolvePassphraseMode(passphraseFlags{}, env, "prompt", true)
	if err != nil {
		t.Fatalf("resolvePassphraseMode: %v", err)
	}
	if mode != passphraseStdin {
		t.Errorf("env+settings+default present, no flag: mode = %v, want %v (env wins)", mode, passphraseStdin)
	}

	// No flag, no env: settings and default contribute; settings wins.
	mode, err = resolvePassphraseMode(passphraseFlags{}, noEnv, "prompt", true)
	if err != nil {
		t.Fatalf("resolvePassphraseMode: %v", err)
	}
	if mode != passphrasePrompt {
		t.Errorf("settings+default present, no flag/env: mode = %v, want %v (settings wins)", mode, passphrasePrompt)
	}

	// Nothing but the built-in default, TTY attached: prompt.
	mode, err = resolvePassphraseMode(passphraseFlags{}, noEnv, "", true)
	if err != nil {
		t.Fatalf("resolvePassphraseMode: %v", err)
	}
	if mode != passphrasePrompt {
		t.Errorf("only the built-in default, TTY attached: mode = %v, want %v", mode, passphrasePrompt)
	}

	// Nothing at all, no TTY: fail closed.
	if _, err := resolvePassphraseMode(passphraseFlags{}, noEnv, "", false); !errors.Is(err, app.ErrUsage) {
		t.Errorf("nothing configured, no TTY: err = %v, want ErrUsage (fail closed, tdd.md §11)", err)
	}
}

func TestResolvePassphraseMode_InvalidEnvValue(t *testing.T) {
	env := func(string) (string, bool) { return "bogus", true }
	if _, err := resolvePassphraseMode(passphraseFlags{}, env, "", true); !errors.Is(err, app.ErrUsage) {
		t.Errorf("invalid env value: err = %v, want ErrUsage", err)
	}
}

func TestResolvePassphraseMode_InvalidSettingsValue(t *testing.T) {
	if _, err := resolvePassphraseMode(passphraseFlags{}, noEnv, "bogus", true); !errors.Is(err, app.ErrUsage) {
		t.Errorf("invalid settings value: err = %v, want ErrUsage", err)
	}
}

func TestObtainPassphrase_None(t *testing.T) {
	b, err := obtainPassphrase(passphraseNone)
	if err != nil {
		t.Fatalf("obtainPassphrase(none): %v", err)
	}
	if b != nil {
		t.Errorf("obtainPassphrase(none) = %v, want nil", b)
	}
}

func TestReadPassphraseLine_StopsAtNewline(t *testing.T) {
	// readPassphraseLine must consume exactly one line, leaving the rest of the reader untouched
	// — new key's own doc comment on the collision this avoids with write.go's confirm prompt.
	r := bytes.NewBufferString("hunter2\nrest-of-stdin")
	got, err := readPassphraseLine(r)
	if err != nil {
		t.Fatalf("readPassphraseLine: %v", err)
	}
	if string(got) != "hunter2" {
		t.Errorf("readPassphraseLine = %q, want %q", got, "hunter2")
	}
	// A second read past the newline sees exactly what is left in the buffer, proving the first
	// read did not consume more than its own line.
	remaining, err := readPassphraseLine(r)
	if err != nil {
		t.Fatalf("readPassphraseLine (second read): %v", err)
	}
	if string(remaining) != "rest-of-stdin" {
		t.Errorf("remaining after one line consumed = %q, want %q", remaining, "rest-of-stdin")
	}
}

func TestReadPassphraseLine_NoTrailingNewline(t *testing.T) {
	r := bytes.NewBufferString("hunter2")
	got, err := readPassphraseLine(r)
	if err != nil {
		t.Fatalf("readPassphraseLine: %v", err)
	}
	if string(got) != "hunter2" {
		t.Errorf("readPassphraseLine = %q, want %q", got, "hunter2")
	}
}

// TestReadPassphraseLine_NoReallocationForRealisticLength is the regression test for the
// reviewer's secret-hygiene finding: line must be pre-sized to maxPassphraseLineLen up front, so a
// realistic passphrase never forces append to grow into a fresh backing array (leaving the old one
// -- still holding a passphrase prefix -- abandoned and never zeroed).
func TestReadPassphraseLine_NoReallocationForRealisticLength(t *testing.T) {
	passphrase := strings.Repeat("x", 200) // hundreds of times longer than any real passphrase
	r := bytes.NewBufferString(passphrase + "\n")

	// append only reallocates once len would exceed cap, so asserting cap(got) equals
	// maxPassphraseLineLen exactly (readPassphraseLine's pre-sized starting capacity) structurally
	// proves no reallocation happened anywhere during the read, without needing unsafe pointer
	// comparisons.
	got, err := readPassphraseLine(r)
	if err != nil {
		t.Fatalf("readPassphraseLine: %v", err)
	}
	if string(got) != passphrase {
		t.Fatalf("readPassphraseLine returned %d bytes, want %d", len(got), len(passphrase))
	}
	if cap(got) != maxPassphraseLineLen {
		t.Errorf("cap(got) = %d, want %d (pre-sized capacity, unchanged by any append-driven reallocation)", cap(got), maxPassphraseLineLen)
	}
}

// TestReadPassphraseLine_ExactlyAtCapAccepted is the boundary case the two neighboring tests
// straddle but never hit directly: exactly maxPassphraseLineLen bytes must succeed (not be
// off-by-one rejected), and the returned slice's capacity must still be exactly the pre-sized
// value, since acceptance at the cap is exactly where a fencepost error in the `>=` check in
// readPassphraseLine would most likely hide.
func TestReadPassphraseLine_ExactlyAtCapAccepted(t *testing.T) {
	passphrase := strings.Repeat("x", maxPassphraseLineLen)
	r := bytes.NewBufferString(passphrase + "\n")

	got, err := readPassphraseLine(r)
	if err != nil {
		t.Fatalf("readPassphraseLine at exactly the cap: %v", err)
	}
	if len(got) != maxPassphraseLineLen {
		t.Fatalf("readPassphraseLine at exactly the cap: len(got) = %d, want %d", len(got), maxPassphraseLineLen)
	}
	if cap(got) != maxPassphraseLineLen {
		t.Errorf("cap(got) = %d, want %d", cap(got), maxPassphraseLineLen)
	}
}

// TestReadPassphraseLine_OverLengthRejectedNotTruncated confirms an input past maxPassphraseLineLen
// is rejected with a clear error — not silently truncated, and not silently grown past the
// pre-sized capacity (which would defeat the no-reallocation property the sibling test above
// checks).
func TestReadPassphraseLine_OverLengthRejectedNotTruncated(t *testing.T) {
	tooLong := strings.Repeat("x", maxPassphraseLineLen+1)
	r := bytes.NewBufferString(tooLong + "\n")

	got, err := readPassphraseLine(r)
	if !errors.Is(err, app.ErrUsage) {
		t.Fatalf("readPassphraseLine over cap: err = %v, want ErrUsage", err)
	}
	if got != nil {
		t.Errorf("readPassphraseLine over cap: got = %v, want nil (rejected, not truncated or returned)", got)
	}
}

func TestZeroBytes(t *testing.T) {
	b := []byte("secret")
	zeroBytes(b)
	for i, v := range b {
		if v != 0 {
			t.Errorf("b[%d] = %v, want 0", i, v)
		}
	}
}
