package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// markerFileName is the marker hasp writes at the leaf of a managed profile directory (D13, D15).
const markerFileName = ".hasp"

// profileMarkerHeader is the `#`-prefixed content adopt profile writes into a freshly created
// .hasp marker (D15 elaboration 3: "`adopt` creates the file with a `#` header stating what the
// file is, that its presence is the signal, and that `#` lines are ignored"). Every line is
// `#`-prefixed on purpose (D15's own rationale: only an actually-`#`-prefixed header makes the
// convention real rather than folklore) so a marker written today parses as zero data to any
// future reader, exactly the forward-compatibility guarantee D15's "Consequence" section states.
const profileMarkerHeader = `# This directory is a hasp-managed profile.
# hasp only checks that this file exists — it never reads or parses what is written below.
# Lines starting with # are ignored by hasp; feel free to leave yourself a note.
`

// CreateMarker marks Dir as a managed profile by writing its .hasp marker (D13, D15). It fails
// closed — ErrMarkerExists — the instant a marker already exists at Dir, before any byte is
// written: a marker may carry a note the user wrote (D15), and CreateMarker never silently
// replaces it. Dir itself must already exist; the mkdir-p half of "new profile"'s grid cell
// (tdd.md §9) belongs to that later use case's own Plan construction, not to this primitive —
// tdd.md §4's type sketch has no distinct "create directory" Change kind, and adding one is out
// of scope for this phase.
type CreateMarker struct {
	Dir    string
	Header []byte
}

func (c CreateMarker) markerPath() string { return filepath.Join(c.Dir, markerFileName) }

func (c CreateMarker) Preview() Preview {
	return Preview{Summary: fmt.Sprintf("mark %s as a managed profile", c.Dir)}
}

// RequiresBackup is false: a marker creation is a pure creation, nothing existed at its path
// before it (Apply fails closed if that's not true), so there is no prior state to preserve.
func (c CreateMarker) RequiresBackup() bool { return false }

func (c CreateMarker) Apply(fsys WriteFS) error {
	path := c.markerPath()
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("%w: %s", ErrMarkerExists, path)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("check %s: %w", path, err)
	}
	return fsys.WriteFile(path, c.Header, 0o644)
}
