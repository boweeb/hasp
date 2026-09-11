// Package render holds hasp's two output forms — human (tabwriter-based) and JSON (a versioned
// envelope) — fed from the same app-layer values (T13: one code path computes an answer, two
// display it), so the two can diverge in formatting but never in content.
package render

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
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
	raw, err := marshalData(data)
	if err != nil {
		return fmt.Errorf("marshal %s data: %w", kind, err)
	}
	env := Envelope{Version: EnvelopeVersion, Kind: kind, Data: raw, Warnings: warnings}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

// marshalData normalizes every nil Go slice reachable inside data to JSON "[]" rather than
// encoding/json's default "null" — at every nesting depth, not only at data itself. Every
// list/find-shaped use case in internal/app returns a bare Go nil slice for "no results"
// (idiomatic Go, and correct — there's nothing wrong with the app layer doing this), but T14
// states the --json pipeline contract in exactly these terms: "hasp list key --json | jq has to
// keep working release over release." `jq '.data[]'` errors on a JSON null; a consumer should
// never have to special-case "no results" apart from "zero results," so this is the one place
// — now genuinely one place, not one place per field — that distinction gets erased before it
// reaches the wire.
//
// T52 replaces this function's original top-level-only special case with a deep walk
// (normalizeNilSlices) after finding it did not, in fact, cover every field: "profiles"
// (domain.Key.Profiles) and "hosts" (app.KeyDetail.Hosts) both reached the wire as a literal
// JSON null, because they are nested one struct-field deep from data rather than being data
// itself. A per-field fix (each producer remembering to return []T{} instead of nil) does not
// scale to every future field a use case might add; a recursive walk fixes the class of defect
// once, here, the same way this function already fixed the top-level case once instead of
// patching every call site. See normalizeNilSlices's own doc comment for the traps a naive
// implementation of this walk would fall into.
func marshalData(data any) ([]byte, error) {
	if data == nil {
		return json.Marshal(data)
	}
	normalized := normalizeNilSlices(reflect.ValueOf(data))
	return json.Marshal(normalized.Interface())
}

// jsonMarshalerType is json.Marshaler's reflect.Type, used to detect a type that owns its own
// wire encoding so normalizeNilSlices never descends into (and never risks corrupting) it.
var jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()

// textMarshalerType is encoding.TextMarshaler's reflect.Type (revision cycle 1, verifier finding
// 1). encoding/json defers to TextMarshaler — encoding the value as a JSON string via
// MarshalText — for any type that does not implement json.Marshaler; trap 3's ownership argument
// (a type that owns its own wire encoding must never be rebuilt field-by-field, because
// reconstruction silently zeroes unexported state via the CanSet() skip in the Struct case, and a
// MarshalText/MarshalJSON method reads the corrupted copy afterward) applies identically whether
// the owning method is named MarshalJSON or MarshalText — encoding/json's choice of which one to
// call is an implementation detail of *how* the type is serialized, not a fact about whether it
// owns that serialization. No type in this tree implements only TextMarshaler as of this writing
// (verified: `grep -rn "MarshalText" internal/` finds nothing), so this guards a class of future
// defect, not a live one — exactly the same posture trap 2's []byte guard already takes.
var textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()

// normalizeNilSlices returns a value equal to v for encoding/json's purposes, except that every
// nil slice reachable inside it (at any depth, through structs, slices, maps, pointers, and
// interfaces) is replaced with a non-nil, zero-length slice of the same type, so it marshals as
// "[]" rather than "null". It never mutates v itself — v (and everything reachable from it)
// belongs to the caller, and marshalData's only contract with internal/app is to render what it
// is given, not to change it out from under a caller that might reuse the same value afterward
// (see TestMarshalData_DoesNotMutateInput).
//
// Four traps make a naive "just walk every slice and fix nils" implementation wrong:
//
//  1. json.RawMessage never appears in this walk. marshalData only ever receives Envelope.Data's
//     *source* value — the app-layer payload JSON() is about to marshal into that field — never
//     the Envelope itself (see JSON above: raw, err := marshalData(data), and Envelope.Data is
//     assigned raw only afterward). A json.RawMessage — itself a []byte, and therefore caught by
//     trap 2 below regardless — is simply never a value this function is ever called with or
//     recurses into.
//
//  2. []byte is not an array-typed field for T14's purposes — it is encoding/json's one
//     documented exception to "slices encode as JSON arrays": encoding/json base64-encodes a
//     []byte (and any named type whose element Kind is Uint8, since encoding/json's own
//     special-case check is by element kind, not by exact type identity) into a JSON *string*,
//     and a nil one into "null" — a bare JSON null, not an empty array, and correctly so:
//     rewriting a nil []byte to a non-nil, zero-length one would flip its wire encoding from
//     "null" to `""`, changing the emitted JSON *type* (string vs. array) for a field T14's
//     contract was never about in the first place. This is the sharpest edge in this function —
//     get the Kind() check wrong (checking the slice's own Kind instead of its element's) and
//     every []byte-shaped field silently breaks. There is no such field in this tree today, but
//     the guard costs nothing and a future one must not regress this.
//
//  3. A type that implements json.Marshaler OR encoding.TextMarshaler owns its output completely
//     — domain.KeyFormat, domain.BindingKind, domain.ProfilePath, domain.Confidence,
//     domain.SchemeID, domain.KeyIdentity's byFingerprint/byPath, app.Severity, and app.DiffKind,
//     as of this writing (T52), all via json.Marshaler; no type in this tree implements only
//     TextMarshaler as of this writing (revision cycle 1, verifier finding 1) — and rebuilding one
//     field-by-field through reflection (dropping its unexported state per trap 5, or simply
//     second-guessing a Marshal method that already produces correct output) is exactly the kind
//     of "helpful" rewrite that can silently change what it emits. Both interfaces matter for
//     identically the same reason, not two different ones: encoding/json defers to TextMarshaler
//     — encoding the value as a JSON string via MarshalText — for any type that does not
//     implement json.Marshaler, so a TextMarshaler-only type owns its wire output exactly as
//     completely as a json.Marshaler one does; checking only json.Marshaler would leave a future
//     TextMarshaler-only type walked, its unexported state zeroed, and its own MarshalText left
//     to read the corrupted copy — the identical failure trap 3 exists to prevent, just reached
//     through the other interface encoding/json actually consults. normalizeNilSlices checks both
//     a value's own type and a pointer to it against both jsonMarshalerType and textMarshalerType
//     (a pointer-receiver Marshal method set only attaches to *T, not T) and, if any of the four
//     checks matches, returns that value completely untouched rather than descending into it.
//     domain.Origin.Because ([]domain.ReasonToken) is explicitly still in scope despite this rule:
//     ReasonToken implements neither interface (it marshals as a plain JSON string, its
//     underlying type, via encoding/json's own default string handling), so the *slice* Because
//     is normalized as an ordinary nil-checked slice — it is the element type owning a custom
//     marshaler that disqualifies descending into *that element*, never the slice merely
//     containing marshaler-owning (or, as here, ordinary) elements.
//
//  4. A map is left alone when nil, never normalized to a non-nil empty map. T14's stated
//     contract — `jq '.data[]'` must not choke on "no results" — is about **arrays**;
//     encoding/json's null-vs-empty distinction for objects is a different question this
//     function was never asked to settle, and the one map in this tree, app.Finding.Detail
//     (map[string]any), is tagged `json:",omitempty"`, so a nil value there never reaches the
//     wire as `"detail": null` in the first place — omitempty drops the field entirely. A map's
//     *values* are still walked and normalized when the map itself is non-nil — Detail's own
//     documented case, `{"paths": [...]}`, stores a []string in a map[string]any value, and that
//     slice is exactly as subject to the nil-to-"[]" rule as any other.
//
// Unexported struct fields are skipped, not zero-valued by omission: a struct field obtained by
// reflection off an unexported field can never be Set (the reflect package's own visibility
// rule), and encoding/json never marshals an unexported field either — the two facts compose to
// make skipping such a field lossless for whatever normalizeNilSlices ultimately hands to
// json.Marshal, not an oversight that happens to look harmless.
//
// Nil pointers, nil interfaces, and nil maps pass through unchanged (only a nil *slice* is ever
// rewritten); a non-nil pointer or interface is walked through to its pointee/dynamic value.
//
// Performance is not a design constraint here (P8 scopes hasp to one laptop and tens of keys,
// never a service processing bulk JSON) — this function allocates a fresh copy of every
// slice/map/struct/pointer/interface it walks through, on every call, rather than mutating or
// sharing anything in place, and that is a deliberate trade for trap-proof correctness (and the
// no-mutation guarantee above) over speed.
func normalizeNilSlices(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	t := v.Type()

	// Trap 3: a type (or a pointer to it) that owns its own wire encoding — via json.Marshaler
	// or encoding.TextMarshaler, encoding/json's fallback when json.Marshaler is absent — is
	// never rebuilt by this walk — return it exactly as the caller supplied it.
	if t.Implements(jsonMarshalerType) || reflect.PointerTo(t).Implements(jsonMarshalerType) ||
		t.Implements(textMarshalerType) || reflect.PointerTo(t).Implements(textMarshalerType) {
		return v
	}

	switch t.Kind() {
	case reflect.Slice:
		// Trap 2: a byte slice (by element Kind, not exact type — encoding/json's own rule)
		// encodes as a base64 string, or "null" when nil; never rewrite it to "[]".
		if t.Elem().Kind() == reflect.Uint8 {
			return v
		}
		if v.IsNil() {
			return reflect.MakeSlice(t, 0, 0)
		}
		out := reflect.MakeSlice(t, v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(normalizeNilSlices(v.Index(i)))
		}
		return out

	case reflect.Array:
		// Fixed-length, never nil, but may still contain a slice-typed element worth
		// normalizing (none exist in this tree as of T52; handled for completeness, at zero
		// extra risk — an array can only ever be walked elementwise, same as a struct).
		out := reflect.New(t).Elem()
		for i := 0; i < t.Len(); i++ {
			out.Index(i).Set(normalizeNilSlices(v.Index(i)))
		}
		return out

	case reflect.Map:
		// Trap 4: a nil map is left as-is; a non-nil map's values are walked and copied into a
		// fresh map so the original's entries are never mutated (no-mutation guarantee above).
		if v.IsNil() {
			return v
		}
		out := reflect.MakeMapWithSize(t, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			out.SetMapIndex(iter.Key(), normalizeNilSlices(iter.Value()))
		}
		return out

	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		out := reflect.New(t.Elem())
		out.Elem().Set(normalizeNilSlices(v.Elem()))
		return out

	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		out := reflect.New(t).Elem()
		out.Set(normalizeNilSlices(v.Elem()))
		return out

	case reflect.Struct:
		out := reflect.New(t).Elem()
		for i := 0; i < t.NumField(); i++ {
			field := out.Field(i)
			// An unexported field can never be Set via reflection — and encoding/json never
			// marshals one either, so skipping it here (leaving out's copy at its zero value)
			// changes nothing about what ultimately reaches the wire.
			if !field.CanSet() {
				continue
			}
			field.Set(normalizeNilSlices(v.Field(i)))
		}
		return out

	default:
		// Scalar kinds (bool, every numeric kind, string) and anything else not handled above
		// carry no nil-slice risk of their own; return as-is.
		return v
	}
}
