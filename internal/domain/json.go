package domain

import (
	"encoding/json"
	"fmt"
)

// marshalStringer is the shared MarshalJSON body for hasp's small enum types: they render as
// their String() form in JSON, exactly as the human renderer prints them, so a finding or field
// seen in the terminal is findable in --json output without translating between two vocabularies
// (tdd.md §10).
func marshalStringer(s fmt.Stringer) ([]byte, error) {
	return json.Marshal(s.String())
}
