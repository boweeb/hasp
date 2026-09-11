package domain

import (
	"encoding/json"
	"testing"
)

// TestAllConfidences_GoldenList is T37's guard test (tdd.md §12, the same style as
// internal/app's AllFindingIDs golden list, T29): the exact four-value confidence set is
// asserted by name and by order, so a fifth value anywhere in the codebase — or a silent
// redefinition of one of the four — shows up in a diff here rather than in a consumer accepting
// a value outside the set tdd.md §16 row 7 declares closed and permanent.
func TestAllConfidences_GoldenList(t *testing.T) {
	want := []Confidence{ConfidenceDerived, ConfidenceConfirmed, ConfidencePossible, ConfidenceUnknown}
	got := AllConfidences()
	if len(got) != len(want) {
		t.Fatalf("AllConfidences() has %d entries, want %d (T37's closed set): got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllConfidences()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	wantStrings := map[Confidence]string{
		ConfidenceDerived:   "derived",
		ConfidenceConfirmed: "confirmed",
		ConfidencePossible:  "possible",
		ConfidenceUnknown:   "unknown",
	}
	for c, s := range wantStrings {
		if c.String() != s {
			t.Errorf("Confidence %v.String() = %q, want %q", c, c.String(), s)
		}
	}
}

// TestSchemeID_ExactStrings pins T36/T35's two verbatim-required ids, plus this chunk's naming
// judgment call for the other two, against unintentional drift.
func TestSchemeID_ExactStrings(t *testing.T) {
	cases := map[SchemeID]string{
		SchemeAWSCreatedRSA:   "aws-created-rsa",
		SchemeAWSImportedRSA:  "aws-imported-rsa",
		SchemeSSHNativeSHA256: "ssh-native-sha256",
		SchemeLegacySSHMD5:    "legacy-ssh-md5",
	}
	for id, want := range cases {
		if string(id) != want {
			t.Errorf("SchemeID %v = %q, want %q", id, string(id), want)
		}
	}
}

// TestAllFactSources_GoldenList discharges FactSource's own doc comment's "forward note, chunk
// M3.6.4": now that FactSource reaches --json (InvestigatedKey.CommentSource,
// internal/app/investigate.go), it gets TestAllConfidences_GoldenList's identical treatment —
// asserted by name and by order, so a third source, or a silent redefinition of one of these two,
// shows up in a diff here rather than reaching a consumer unannounced.
func TestAllFactSources_GoldenList(t *testing.T) {
	want := []FactSource{FactSourcePlainRead, FactSourceAgent}
	got := AllFactSources()
	if len(got) != len(want) {
		t.Fatalf("AllFactSources() has %d entries, want %d: got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllFactSources()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	wantStrings := map[FactSource]string{
		FactSourcePlainRead: "",
		FactSourceAgent:     "agent-sourced",
	}
	for s, want := range wantStrings {
		if string(s) != want {
			t.Errorf("FactSource %v = %q, want %q", s, string(s), want)
		}
	}
}

// TestOrigin_JSONTags pins T37's literal worked example ({"id": ..., "confidence": ...,
// "because": [...]}) against a struct-tag typo silently changing the wire shape.
func TestOrigin_JSONTags(t *testing.T) {
	o := Origin{
		ID:         OriginAWSEC2Created,
		Confidence: ConfidencePossible,
		Because:    []ReasonToken{"algorithm=rsa", "format=pem", ReasonNoConsoleFingerprintSupplied},
	}
	b, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("json.Marshal(Origin): %v", err)
	}
	want := `{"id":"aws-ec2-created","confidence":"possible","because":["algorithm=rsa","format=pem","no-console-fingerprint-supplied"]}`
	if string(b) != want {
		t.Errorf("Origin JSON shape =\n  %s\nwant\n  %s", b, want)
	}
}
