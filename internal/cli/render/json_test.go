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

// --- T52: normalizeNilSlices's deep-walk unit tests, one per trap its own doc comment names. ---

// nestedSliceHolder is a struct one level deep — the exact shape domain.Key.Profiles and
// app.KeyDetail.Hosts actually are (a nil slice nested inside a struct field), which the
// pre-T52 top-level-only special case could not reach at all.
type nestedSliceHolder struct {
	Items []string `json:"items"`
}

type nestedSliceWrapper struct {
	Nested nestedSliceHolder `json:"nested"`
}

// TestNormalizeNilSlices_NestedFieldNormalized proves the whole point of T52: a nil slice one
// struct field deep — unreachable by the pre-T52 top-level-only check — now marshals as "[]".
func TestNormalizeNilSlices_NestedFieldNormalized(t *testing.T) {
	raw, err := marshalData(nestedSliceWrapper{Nested: nestedSliceHolder{Items: nil}})
	if err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	if !bytes.Contains(raw, []byte(`"items":[]`)) {
		t.Errorf("raw = %s, want \"items\":[]", raw)
	}
	if bytes.Contains(raw, []byte(`"items":null`)) {
		t.Errorf("raw = %s, want no \"items\":null", raw)
	}
}

// TestNormalizeNilSlices_SliceOfStructsEachNormalized proves normalization reaches every
// element of a slice, not merely the first, and each element's own nested nil slice is
// normalized independently.
func TestNormalizeNilSlices_SliceOfStructsEachNormalized(t *testing.T) {
	in := []nestedSliceHolder{{Items: nil}, {Items: []string{"a"}}, {Items: nil}}
	raw, err := marshalData(in)
	if err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	want := `[{"items":[]},{"items":["a"]},{"items":[]}]`
	if string(raw) != want {
		t.Errorf("raw = %s, want %s", raw, want)
	}
}

// byteSliceHolder covers trap 2 (the sharpest edge in normalizeNilSlices): []byte is
// encoding/json's one documented exception — it base64-encodes, and a nil one is "null", never
// "[]". Rewriting it would flip the emitted JSON's *type*, not merely its value.
type byteSliceHolder struct {
	NilBytes    []byte `json:"nilBytes"`
	NonNilBytes []byte `json:"nonNilBytes"`
}

func TestNormalizeNilSlices_ByteSliceLeftAlone(t *testing.T) {
	raw, err := marshalData(byteSliceHolder{NilBytes: nil, NonNilBytes: []byte("hi")})
	if err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	if !bytes.Contains(raw, []byte(`"nilBytes":null`)) {
		t.Errorf("raw = %s, want \"nilBytes\":null (never \"[]\") — []byte is base64/null, not an array", raw)
	}
	if bytes.Contains(raw, []byte(`"nilBytes":[]`)) {
		t.Errorf("raw = %s, a nil []byte must never be rewritten to \"[]\"", raw)
	}
	// "hi" base64-encodes to "aGk=".
	if !bytes.Contains(raw, []byte(`"nonNilBytes":"aGk="`)) {
		t.Errorf("raw = %s, want \"nonNilBytes\":\"aGk=\" (base64), not an array of numbers", raw)
	}
}

// marshalerWithNullSlice implements json.Marshaler and deliberately emits a literal JSON null
// for what looks, from the outside, exactly like the "nested nil slice" shape this function
// otherwise fixes — proving trap 3: a type owning its own MarshalJSON is never rebuilt or
// second-guessed by the walk, even when doing so would "improve" its output.
type marshalerWithNullSlice struct {
	items []string // unexported: MarshalJSON's use of it, not reflection's, is what must be tested
}

func (m marshalerWithNullSlice) MarshalJSON() ([]byte, error) {
	return []byte(`{"items":null}`), nil
}

type marshalerWrapper struct {
	Custom marshalerWithNullSlice `json:"custom"`
}

func TestNormalizeNilSlices_NeverDescendsIntoJSONMarshaler(t *testing.T) {
	raw, err := marshalData(marshalerWrapper{Custom: marshalerWithNullSlice{items: nil}})
	if err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	want := `{"custom":{"items":null}}`
	if string(raw) != want {
		t.Errorf("raw = %s, want %s (the type's own MarshalJSON output, untouched)", raw, want)
	}
}

// TestNormalizeNilSlices_NilPointer proves rule 7: a nil pointer is left nil (it names a single
// optional value, not a "no results" collection — T14's contract is about arrays) rather than
// being treated as anything slice-shaped.
func TestNormalizeNilSlices_NilPointer(t *testing.T) {
	type ptrWrapper struct {
		Ptr *nestedSliceHolder `json:"ptr"`
	}
	raw, err := marshalData(ptrWrapper{Ptr: nil})
	if err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	if want := `{"ptr":null}`; string(raw) != want {
		t.Errorf("raw = %s, want %s", raw, want)
	}
}

// TestNormalizeNilSlices_NonNilPointerStillNormalized proves the walk continues through a
// non-nil pointer to its pointee's own nested nil slice.
func TestNormalizeNilSlices_NonNilPointerStillNormalized(t *testing.T) {
	type ptrWrapper struct {
		Ptr *nestedSliceHolder `json:"ptr"`
	}
	raw, err := marshalData(ptrWrapper{Ptr: &nestedSliceHolder{Items: nil}})
	if err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	if want := `{"ptr":{"items":[]}}`; string(raw) != want {
		t.Errorf("raw = %s, want %s", raw, want)
	}
}

// TestNormalizeNilSlices_NilInterface proves rule 7's other half: a nil `any`/interface field
// (map[string]any's own value type in app.Finding.Detail) is left nil, not panicked on.
func TestNormalizeNilSlices_NilInterface(t *testing.T) {
	type ifaceWrapper struct {
		Any any `json:"any"`
	}
	raw, err := marshalData(ifaceWrapper{Any: nil})
	if err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	if want := `{"any":null}`; string(raw) != want {
		t.Errorf("raw = %s, want %s", raw, want)
	}
}

// TestNormalizeNilSlices_MapValueNilSliceNormalized covers trap 4's map-values half: T29's
// documented app.Finding.Detail shape, map[string]any whose value holds a nil []string
// (`{"paths": [...]}`), has that value normalized exactly like any other reachable nil slice.
func TestNormalizeNilSlices_MapValueNilSliceNormalized(t *testing.T) {
	var nilPaths []string
	in := map[string]any{"paths": nilPaths}
	raw, err := marshalData(in)
	if err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	if want := `{"paths":[]}`; string(raw) != want {
		t.Errorf("raw = %s, want %s", raw, want)
	}
}

// TestNormalizeNilSlices_NilMapLeftAsIs covers trap 4's other half: a nil map itself is never
// rewritten to a non-nil empty map — T14's contract is about arrays, and app.Finding.Detail
// (the one map in this tree) is tagged omitempty, so a nil map here never reaches the wire at
// all in practice; this test pins that a bare (non-omitempty) nil map still marshals as the
// ordinary JSON null encoding/json already gives it, untouched by this function.
func TestNormalizeNilSlices_NilMapLeftAsIs(t *testing.T) {
	type mapWrapper struct {
		Detail map[string]any `json:"detail"`
	}
	raw, err := marshalData(mapWrapper{Detail: nil})
	if err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	if want := `{"detail":null}`; string(raw) != want {
		t.Errorf("raw = %s, want %s (a nil map is left as-is, deliberately not normalized)", raw, want)
	}
}

// TestMarshalData_DoesNotMutateInput is constraint 4's own proof: normalizeNilSlices must work
// on a fresh copy at every level it recurses into, never writing through the caller's own
// pointer, slice backing array, or map. A pointer is the sharpest test of this — if
// normalizeNilSlices ever wrote its normalized result back through v.Elem().Set(...) instead of
// building a brand-new pointee, the caller's own struct would observe its nil slice silently
// replaced by a non-nil empty one after a call that is supposed to be read-only.
func TestMarshalData_DoesNotMutateInput(t *testing.T) {
	target := &nestedSliceHolder{Items: nil}
	if _, err := marshalData(target); err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	if target.Items != nil {
		t.Fatalf("marshalData mutated the caller's value through its pointer: Items = %#v, want nil (unchanged)", target.Items)
	}

	// A slice of structs is the other shape worth proving: normalizing element 0's nil Tags
	// must not touch element 1's already-non-nil Tags, and must not touch either element's
	// field in the *original* backing array at all — the returned slice must be a fresh
	// allocation, not the same backing array with entries rewritten in place.
	orig := []nestedSliceHolder{{Items: nil}, {Items: []string{"a"}}}
	raw, err := marshalData(orig)
	if err != nil {
		t.Fatalf("marshalData: %v", err)
	}
	if orig[0].Items != nil {
		t.Errorf("marshalData mutated orig[0].Items = %#v, want nil (unchanged)", orig[0].Items)
	}
	if len(orig[1].Items) != 1 || orig[1].Items[0] != "a" {
		t.Errorf("marshalData mutated orig[1].Items = %#v, want [\"a\"] (unchanged)", orig[1].Items)
	}
	if want := `[{"items":[]},{"items":["a"]}]`; string(raw) != want {
		t.Errorf("raw = %s, want %s", raw, want)
	}
}
