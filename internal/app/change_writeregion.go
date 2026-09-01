package app

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// RegionID names which marked region within File a WriteRegion targets, for Preview's summary and
// for a future use case's own bookkeeping. Apply itself does not interpret it — Before/After
// carry everything Apply needs to locate and replace the actual span (see spliceRegion).
type RegionID string

// WriteRegion rewrites one hasp-owned marked region wholesale (D7 elaboration 1, "rigid
// regeneration"). Before is the region's current rendered bytes, captured as a read when the Plan
// was built (P5), so Preview() can produce a real diff (T26); After is the fully regenerated
// replacement. This also covers whole-file-owned creation (D7's custom-host-group case, M3): a
// group file hasp is creating outright has no "current bytes" to read at all, which Apply treats
// identically to the first-write-into-an-existing-file case below (empty Before, ENOENT).
type WriteRegion struct {
	File   string
	Marker RegionID
	Before []byte
	After  []byte
}

func (c WriteRegion) Preview() Preview {
	return Preview{
		Summary: fmt.Sprintf("rewrite region %q in %s", c.Marker, c.File),
		Diff:    lineDiff(c.Before, c.After),
	}
}

// RequiresBackup is always true: WriteRegion replaces existing bytes of a file hasp did not
// necessarily create outright (the co-owned ~/.ssh/config case, D7's harder case), which is
// exactly what P4 exists to protect.
func (c WriteRegion) RequiresBackup() bool { return true }

func (c WriteRegion) backupPath() string { return c.File }

func (c WriteRegion) Apply(fsys WriteFS) error {
	current, err := os.ReadFile(c.File)
	if err != nil {
		// A brand-new group file (D7's whole-file-owned case) doesn't exist on disk yet — there is
		// no prior region to locate, exactly the same "nothing to splice against" shape
		// spliceRegion already handles for an existing-but-empty-region file, just extended to a
		// not-yet-existing file. Any other read error still propagates unchanged.
		if errors.Is(err, fs.ErrNotExist) && len(c.Before) == 0 {
			current = nil
		} else {
			return fmt.Errorf("write region: read %s: %w", c.File, err)
		}
	}
	next, err := spliceRegion(current, c.Before, c.After)
	if err != nil {
		return fmt.Errorf("write region: %s: %w", c.File, err)
	}
	// mode 0: when c.File already exists, WriteFile ignores the mode argument and preserves the
	// file's own current mode instead (§11 point 3). When it does not yet exist (the brand-new
	// group file case above), WriteFile's own default (0o600) applies — a hasp-owned group file
	// carries no secrets, but 0o600 is a harmless, conservative default for a freshly created
	// config file, matching the private-key default rather than the world-readable 0o644 `new key`
	// uses for a .pub sidecar.
	return fsys.WriteFile(c.File, next, 0)
}

// spliceRegion reconstructs the file's full replacement content. WriteRegion carries only the
// region's own bytes (Before/After), not the file's full content, so Apply locates that exact
// span within current and substitutes it. An empty Before means there is nothing to locate — After
// is appended to the end of current instead — which covers two distinct callers identically: a
// first-time region write into an existing file (current holds whatever pre-existed outside the
// region), and a brand-new whole-file-owned group file (current is nil, since Apply never even
// read a file; the "append" degenerates to After becoming the entire new file).
//
// current is guaranteed, by Applier's witness re-verification (T30), to be byte-identical to what
// Plan() read when it captured Before — so an exact, single-occurrence match is expected, not a
// best-effort search. A missing or non-unique match means Before does not actually describe a
// span of current, which spliceRegion refuses to guess about rather than silently mangling the
// file.
func spliceRegion(current, before, after []byte) ([]byte, error) {
	if len(before) == 0 {
		out := make([]byte, 0, len(current)+len(after))
		out = append(out, current...)
		out = append(out, after...)
		return out, nil
	}

	idx := bytes.Index(current, before)
	if idx < 0 {
		return nil, fmt.Errorf("region bytes not found in current content")
	}
	if bytes.Contains(current[idx+1:], before) {
		return nil, fmt.Errorf("region bytes are not unique in current content, refusing to guess which occurrence to replace")
	}

	out := make([]byte, 0, len(current)-len(before)+len(after))
	out = append(out, current[:idx]...)
	out = append(out, after...)
	out = append(out, current[idx+len(before):]...)
	return out, nil
}
