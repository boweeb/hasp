// Package render holds hasp's two output forms — human (tabwriter-based) and JSON (a versioned
// envelope) — fed from the same app-layer values (T13: one code path computes an answer, two
// display it), so the two can diverge in formatting but never in content.
package render

import (
	"encoding/json"
	"fmt"
	"io"
)

// EnvelopeVersion is the JSON envelope's own schema version (T14) — independent of hasp's
// release version. It moves only if the envelope's *shape* changes; adding a field to Data does
// not require a bump.
const EnvelopeVersion = 1

// Envelope wraps every --json response (tdd.md §10).
type Envelope struct {
	Version  int             `json:"version"`
	Kind     string          `json:"kind"` // "key.list", "host.show", "check.report", ...
	Data     json.RawMessage `json:"data"`
	Warnings []string        `json:"warnings,omitempty"`
}

// JSON marshals data into data's Envelope and writes it to w, pretty-printed.
func JSON(w io.Writer, kind string, data any, warnings []string) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal %s data: %w", kind, err)
	}
	env := Envelope{Version: EnvelopeVersion, Kind: kind, Data: raw, Warnings: warnings}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}
