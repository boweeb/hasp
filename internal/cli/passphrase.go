// passphrase.go implements `new key`'s passphrase mode resolution (tdd.md §9, T6, D17): the
// flag -> env -> settings -> built-in-default chain, owned directly rather than via spf13/viper
// (T7's explicit non-dependency) because it is small enough to keep the precedence rule auditable
// in one place. internal/app never sees an env var, a TTY, or a settings file — only the decided
// mode and the resulting secret bytes (tdd.md §3's Request-only framing).
package cli

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/boweeb/hasp/internal/app"
)

// passphraseEnvVar is T6's env var: HASP_NEW_KEY_PASSPHRASE_MODE, one of none/prompt/stdin.
const passphraseEnvVar = "HASP_NEW_KEY_PASSPHRASE_MODE"

type passphraseMode string

const (
	passphraseNone   passphraseMode = "none"
	passphrasePrompt passphraseMode = "prompt"
	passphraseStdin  passphraseMode = "stdin"
)

func parsePassphraseMode(s string) (passphraseMode, bool) {
	switch passphraseMode(s) {
	case passphraseNone, passphrasePrompt, passphraseStdin:
		return passphraseMode(s), true
	default:
		return "", false
	}
}

// passphraseFlags is `new key`'s three mutually exclusive flags (tdd.md §9).
type passphraseFlags struct {
	Passphrase      bool
	NoPassphrase    bool
	PassphraseStdin bool
}

// explicit returns the mode named by exactly one set flag. More than one set is a usage error
// (T6: mutually exclusive, exit code 2); none set reports ok=false so the caller falls through to
// the resolution chain's next tier.
func (f passphraseFlags) explicit() (passphraseMode, bool, error) {
	var mode passphraseMode
	count := 0
	if f.Passphrase {
		mode, count = passphrasePrompt, count+1
	}
	if f.NoPassphrase {
		mode, count = passphraseNone, count+1
	}
	if f.PassphraseStdin {
		mode, count = passphraseStdin, count+1
	}
	if count > 1 {
		return "", false, fmt.Errorf("%w: --passphrase, --no-passphrase, and --passphrase-stdin are mutually exclusive", app.ErrUsage)
	}
	return mode, count == 1, nil
}

// resolvePassphraseMode walks T6's precedence chain: flag -> env -> settings -> built-in default
// ("prompt if a TTY is attached, else fail closed"). lookupEnv and isTTY are injected so this
// resolver — deliberately not spf13/viper, T6/T7 — is fully unit-testable without a real
// environment or terminal.
func resolvePassphraseMode(flags passphraseFlags, lookupEnv func(string) (string, bool), settingsMode string, isTTY bool) (passphraseMode, error) {
	if mode, ok, err := flags.explicit(); err != nil {
		return "", err
	} else if ok {
		return mode, nil
	}

	if raw, ok := lookupEnv(passphraseEnvVar); ok && raw != "" {
		mode, ok := parsePassphraseMode(raw)
		if !ok {
			return "", fmt.Errorf("%w: %s=%q is not one of none/prompt/stdin", app.ErrUsage, passphraseEnvVar, raw)
		}
		return mode, nil
	}

	if settingsMode != "" {
		mode, ok := parsePassphraseMode(settingsMode)
		if !ok {
			return "", fmt.Errorf("%w: settings [new_key] default_passphrase_mode = %q is not one of none/prompt/stdin", app.ErrUsage, settingsMode)
		}
		return mode, nil
	}

	if isTTY {
		return passphrasePrompt, nil
	}
	return "", fmt.Errorf(
		"%w: no passphrase mode was chosen for a non-interactive run — pass --passphrase, --no-passphrase, or --passphrase-stdin, set %s, or configure [new_key] default_passphrase_mode in settings",
		app.ErrUsage, passphraseEnvVar,
	)
}

// obtainPassphrase reads the secret bytes for mode. --passphrase-stdin reads exactly one line
// from stdin, never to EOF, so a later interactive confirm prompt in the same invocation
// (write.go's confirmApply) still has stdin available — this is the collision T6's own flag
// description warns about, avoided by construction rather than by convention.
func obtainPassphrase(mode passphraseMode) ([]byte, error) {
	switch mode {
	case passphraseNone:
		return nil, nil
	case passphrasePrompt:
		return promptPassphrase()
	case passphraseStdin:
		return readPassphraseLine(os.Stdin)
	default:
		return nil, fmt.Errorf("%w: unknown passphrase mode %q", app.ErrUsage, mode)
	}
}

// promptPassphrase reads one line from the real terminal fd with no local echo
// (golang.org/x/term.ReadPassword) — T6's interactive path.
func promptPassphrase() ([]byte, error) {
	fmt.Fprint(os.Stderr, "Passphrase: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("read passphrase: %w", err)
	}
	return b, nil
}

// maxPassphraseLineLen bounds readPassphraseLine's input (D17, T6). It exists for two reasons:
// first, so a runaway or hostile stdin can't grow line without limit; second, and the reason it's
// sized at 4096, it lets readPassphraseLine pre-allocate line's backing array once, up front, at
// exactly this capacity — a passphrase of any realistic length (this is 4096 bytes, hundreds of
// times longer than any passphrase a human would type or a password manager would generate) then
// never forces append to reallocate and copy into a new backing array, which would otherwise leave
// the old array — holding a still-unzeroed prefix of the secret — abandoned in memory with nothing
// in this code ever zeroing it. The caller's defer zeroBytes(secret) only ever reaches the single,
// final backing array this way. This is a residue-minimization measure, not an absolute guarantee:
// it covers the realistic case; it does not by itself prevent the underlying array from being
// copied by the Go runtime's own stack growth or GC moves, which are outside this code's control.
const maxPassphraseLineLen = 4096

// readPassphraseLine reads exactly one line from r, byte by byte rather than through a bufio
// reader — a bufio reader's own internal buffer can pull far more than one line out of the
// underlying fd in a single fill(), and any bytes past the line stay trapped in that buffer,
// invisible to write.go's confirmApply's own, separate read of the same os.Stdin later in the
// same invocation. Reading one byte at a time is the only way to guarantee this stops exactly at
// the delimiter and leaves everything after it untouched on the underlying reader.
//
// line is pre-sized to maxPassphraseLineLen's capacity so that, for any realistic passphrase,
// append never triggers a reallocation — see maxPassphraseLineLen's own comment for why that
// matters for secret hygiene (T6/D17). An input that would exceed the cap is rejected outright,
// not silently truncated or grown past it.
func readPassphraseLine(r io.Reader) ([]byte, error) {
	line := make([]byte, 0, maxPassphraseLineLen)
	one := make([]byte, 1)
	for {
		n, err := r.Read(one)
		if n > 0 {
			if one[0] == '\n' {
				break
			}
			if len(line) >= maxPassphraseLineLen {
				zeroBytes(line)
				return nil, fmt.Errorf("%w: passphrase from stdin exceeds %d bytes", app.ErrUsage, maxPassphraseLineLen)
			}
			line = append(line, one[0])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			zeroBytes(line)
			return nil, fmt.Errorf("read passphrase from stdin: %w", err)
		}
	}
	if n := len(line); n > 0 && line[n-1] == '\r' {
		line = line[:n-1]
	}
	return line, nil
}

// zeroBytes overwrites b's contents in place — explicit, rather than left for garbage collection
// (D17, T6's "buffer zeroed immediately after use"). keyfile.Generate already zeroes the same
// backing array as a side effect of consuming req.Passphrase; this is called defensively at the
// CLI layer too, for the passphraseNone case (a no-op) and as a second line of defense for every
// other case.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
