package settings

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// dirName and fileName together make up tdd.md §8's location:
// os.UserConfigDir()/hasp/settings.toml — stdlib only, no XDG-paths library (T7's explicit
// non-dependency), deliberately outside ~/.ssh.
const (
	dirName  = "hasp"
	fileName = "settings.toml"
)

// NewKeySettings is the v1 (and, at this milestone, only) settings section — new key's
// non-interactive default passphrase mode (tdd.md §8, T6, T7). DefaultPassphraseMode is left as
// a bare string here rather than a typed enum: this package only decodes and hands the raw value
// back, never validates or interprets it — that is internal/cli's job (T6's resolution chain),
// so this package stays free of any opinion about what "none"/"prompt"/"stdin" mean.
type NewKeySettings struct {
	DefaultPassphraseMode string `toml:"default_passphrase_mode"`
}

// Settings is the entire settings.toml schema. The admission rule (D16, T7) is that this file
// may hold only user preferences that (a) cannot be derived from the machine and (b) exist to
// enable non-interactive execution — nothing else may be added to this struct without clearing
// that bar independently. docs/tdd.md §12 requires a test asserting this exact field set by name.
type Settings struct {
	NewKey NewKeySettings `toml:"new_key"`
}

// Path returns the default settings file location, tdd.md §8: os.UserConfigDir()/hasp/settings.toml.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve settings directory: %w", err)
	}
	return filepath.Join(dir, dirName, fileName), nil
}

// Load reads and decodes the settings file at path. hasp reads settings and never writes them
// (D16) — there is no companion Save. A missing file is not an error: it returns the zero-value
// Settings{}, whose every field being its zero value means "not configured," so a caller's own
// resolution chain (T6) falls through to its next tier, preserving tdd.md §6.3's "works on first
// run, nothing configured." A file that exists but does not parse as valid TOML, or whose shape
// does not match Settings, is a real decode error — the settings file is user-authored, and a
// syntax mistake in it should be reported, not silently ignored.
func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Settings{}, nil
		}
		return Settings{}, fmt.Errorf("read settings %s: %w", path, err)
	}

	var s Settings
	if err := toml.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("parse settings %s: %w", path, err)
	}
	return s, nil
}
