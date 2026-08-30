package render

import (
	"bytes"
	"strings"
	"testing"
)

// TestJSON_NilSliceMarshalsAsEmptyArray guards T14's stated pipeline contract ("hasp list key
// --json | jq has to keep working release over release"): a nil Go slice — the ordinary shape
// of "no results" from a list/find use case — must serialize as JSON "[]", never "null", or
// `jq '.data[]'` errors instead of iterating zero times. Found by an independent code review of
// M1 (a nil-slice-to-null bug affecting every check/find detector and any list scoped to zero
// results).
func TestJSON_NilSliceMarshalsAsEmptyArray(t *testing.T) {
	var nilSlice []string
	var buf bytes.Buffer
	if err := JSON(&buf, "test.kind", nilSlice, nil); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !strings.Contains(buf.String(), `"data": []`) {
		t.Errorf("output = %s, want data: []", buf.String())
	}
	if strings.Contains(buf.String(), `"data": null`) {
		t.Errorf("output = %s, want no null data", buf.String())
	}
}

func TestJSON_NonNilEmptySliceStillEmptyArray(t *testing.T) {
	nonNil := []string{}
	var buf bytes.Buffer
	if err := JSON(&buf, "test.kind", nonNil, nil); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !strings.Contains(buf.String(), `"data": []`) {
		t.Errorf("output = %s, want data: []", buf.String())
	}
}

func TestJSON_StructDataUnaffected(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	var buf bytes.Buffer
	if err := JSON(&buf, "test.kind", payload{Name: "id_ed25519"}, nil); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !strings.Contains(buf.String(), `"name": "id_ed25519"`) {
		t.Errorf("output = %s, want the struct's field", buf.String())
	}
}
