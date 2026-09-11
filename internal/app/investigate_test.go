package app

import (
	"path/filepath"
	"testing"

	"github.com/boweeb/hasp/internal/adapter/fpscheme"
	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/adapter/sshagent"
	"github.com/boweeb/hasp/internal/domain"
)

// TestInvestigateKeys_EveryRegisteredScheme is roadmap.md §5.6 exit criterion 2, exercised
// directly against the use case rather than only through the CLI: every registered scheme must
// appear for every key, asserted against len(fpscheme.Schemes()) so a fifth scheme cannot be
// silently dropped from InvestigatedKey.Schemes.
func TestInvestigateKeys_EveryRegisteredScheme(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "rsa-pem-plain-pub", filepath.Join(dir, "id_rsa"))
	copyFixture(t, "rsa-pem-plain-pub.pub", filepath.Join(dir, "id_rsa.pub"))
	copyFixture(t, "ed25519-openssh-plain-pub", filepath.Join(dir, "id_ed25519"))
	copyFixture(t, "ed25519-openssh-plain-pub.pub", filepath.Join(dir, "id_ed25519.pub"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}

	investigated := InvestigateKeys(m, InvestigateRequest{})
	if len(investigated) != len(m.Keys) {
		t.Fatalf("InvestigateKeys returned %d keys, want %d", len(investigated), len(m.Keys))
	}
	wantSchemeCount := len(fpscheme.Schemes())
	for _, k := range investigated {
		if len(k.Schemes) != wantSchemeCount {
			t.Errorf("%s: %d schemes, want %d (one per registered scheme)", k.Name, len(k.Schemes), wantSchemeCount)
		}
		for _, sf := range k.Schemes {
			if sf.Value == "" && sf.Reason == "" {
				t.Errorf("%s/%s: neither a value nor a reason — an unknown scheme must always carry a machine-readable reason", k.Name, sf.Scheme)
			}
		}
	}
}

// TestInvestigateKeys_ConfidenceAndFactSourceInClosedSets is roadmap.md §5.6 exit criterion 3,
// asserted structurally against every value this package itself produces (the CLI-level test,
// internal/cli, additionally walks the --json wire form end to end).
func TestInvestigateKeys_ConfidenceAndFactSourceInClosedSets(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "rsa-pem-plain-pub", filepath.Join(dir, "id_rsa"))
	copyFixture(t, "rsa-pem-plain-pub.pub", filepath.Join(dir, "id_rsa.pub"))
	copyFixture(t, "ed25519-openssh-plain-pub", filepath.Join(dir, "id_ed25519"))
	copyFixture(t, "ed25519-openssh-plain-pub.pub", filepath.Join(dir, "id_ed25519.pub"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}

	confidences := map[domain.Confidence]bool{}
	for _, c := range domain.AllConfidences() {
		confidences[c] = true
	}
	factSources := map[domain.FactSource]bool{}
	for _, s := range domain.AllFactSources() {
		factSources[s] = true
	}

	for _, k := range InvestigateKeys(m, InvestigateRequest{}) {
		for _, sf := range k.Schemes {
			if !confidences[sf.Confidence] {
				t.Errorf("%s/%s: Confidence = %q, not in domain.AllConfidences()", k.Name, sf.Scheme, sf.Confidence)
			}
		}
		for _, o := range k.Origins {
			if !confidences[o.Confidence] {
				t.Errorf("%s origin %s: Confidence = %q, not in domain.AllConfidences()", k.Name, o.ID, o.Confidence)
			}
		}
		if !factSources[k.CommentSource] {
			t.Errorf("%s: CommentSource = %q, not in domain.AllFactSources()", k.Name, k.CommentSource)
		}
	}
}

// TestOriginsForInvestigate_RSA_BothOriginsPossible pins the RSA branch of the M3.6.4-decided
// origin rule: both AWS-EC2 origins, each ConfidencePossible (never confirmed — no external clue
// was ever supplied), each carrying the same three because tokens.
func TestOriginsForInvestigate_RSA_BothOriginsPossible(t *testing.T) {
	k := domain.Key{Algorithm: "ssh-rsa", Format: domain.FormatPKCS1}
	origins := originsForInvestigate(k)
	if len(origins) != 2 {
		t.Fatalf("len(origins) = %d, want 2", len(origins))
	}
	wantIDs := map[string]bool{domain.OriginAWSEC2Created: false, domain.OriginAWSEC2Imported: false}
	for _, o := range origins {
		if o.Confidence != domain.ConfidencePossible {
			t.Errorf("origin %s: Confidence = %q, want possible", o.ID, o.Confidence)
		}
		if _, ok := wantIDs[o.ID]; !ok {
			t.Errorf("unexpected origin id %q", o.ID)
		}
		wantIDs[o.ID] = true
		wantBecause := []domain.ReasonToken{"algorithm=ssh-rsa", "format=pkcs1", domain.ReasonNoConsoleFingerprintSupplied}
		if len(o.Because) != len(wantBecause) {
			t.Fatalf("origin %s: Because = %v, want %v", o.ID, o.Because, wantBecause)
		}
		for i := range wantBecause {
			if o.Because[i] != wantBecause[i] {
				t.Errorf("origin %s: Because[%d] = %q, want %q", o.ID, i, o.Because[i], wantBecause[i])
			}
		}
	}
	for id, seen := range wantIDs {
		if !seen {
			t.Errorf("origin %q missing", id)
		}
	}
}

// TestOriginsForInvestigate_NonRSA_Empty pins the non-RSA branch: an empty array, never an
// invented origin (design.md §5.1 / D12) — neither AWS RSA scheme applies to a non-RSA key at
// all. As of T52, "empty" at this layer is a bare Go `nil` (len(origins) == 0 either way) — see
// TestOriginsForInvestigate_NonRSA_ReturnsIdiomaticNil below for why that is now the right,
// idiomatic answer rather than a `[]domain.Origin{}` this function used to have to construct by
// hand.
func TestOriginsForInvestigate_NonRSA_Empty(t *testing.T) {
	k := domain.Key{Algorithm: "ssh-ed25519", Format: domain.FormatOpenSSH}
	if origins := originsForInvestigate(k); len(origins) != 0 {
		t.Errorf("originsForInvestigate(ed25519) = %v, want empty", origins)
	}
}

// TestOriginsForInvestigate_NonRSA_ReturnsIdiomaticNil is T52's own ripple on this file:
// originsForInvestigate used to return `[]domain.Origin{}` on the non-RSA branch specifically
// because internal/cli/render/json.go's marshalData only normalized a nil slice to "[]" for the
// top-level `data` value it was handed, never for a field nested inside a struct (InvestigatedKey
// .Origins is exactly such a field) — see originsForInvestigate's own doc comment, current
// revision, for the full argument. T52 made marshalData recurse into every nested field, so that
// hand-maintained special case is no longer load-bearing; this test pins the resulting,
// idiomatic-Go shape at this layer (a bare `nil`) so a future edit does not "restore" the old
// `[]domain.Origin{}` construction under the mistaken belief it is still required. The wire-level
// guarantee this used to protect directly — "origins" is never JSON null — is now proven one
// layer up, where it actually lives: internal/cli/render's own unit tests (T52,
// TestNormalizeNilSlices_NestedFieldNormalized and friends) prove the renderer's general
// guarantee, and internal/cli's end-to-end guard
// (TestJSONKinds_NeverEmitArrayTypedNull, internal/cli/jsonnullguard_test.go) proves it holds
// through the full --json pipeline for this kind specifically — key.list/key.show under
// --investigate are among the nine of that guard's twelve enumerated kinds that genuinely carry a
// nested array field and therefore genuinely bite. T53 records the qualification the bare phrase
// "every kind" elided: profile.list and profile.find are structural no-ops for this regression
// class (their payload types carry no nested array field at all), and check.report's own
// map-nested case cannot be driven nil by any live detector. T53's Context enumerates two sites
// carrying that unqualified phrasing; this comment was a third it did not count.
func TestOriginsForInvestigate_NonRSA_ReturnsIdiomaticNil(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, "ed25519-openssh-plain-pub", filepath.Join(dir, "id_ed25519"))
	copyFixture(t, "ed25519-openssh-plain-pub.pub", filepath.Join(dir, "id_ed25519.pub"))
	copyFixture(t, "rsa-pem-encrypted-nopub", filepath.Join(dir, "id_rsa_legacy"))

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Keys) != 2 {
		t.Fatalf("len(m.Keys) = %d, want 2", len(m.Keys))
	}

	for _, k := range InvestigateKeys(m, InvestigateRequest{}) {
		if k.Algorithm == "ssh-rsa" {
			t.Fatalf("test fixture setup produced an RSA key (%s) — this test is only meaningful for the non-RSA branch", k.Name)
		}
		if k.Origins != nil {
			t.Errorf("%s: Origins = %#v, want nil (idiomatic Go zero value, T52)", k.Name, k.Origins)
		}
	}
}

// TestRefineSchemeReasons_ReplacesNoTTYReason is roadmap.md §5.6 exit criterion 5's precision
// requirement: fpscheme.ComputeAll's generic ReasonPrivateKeyUnavailable for aws-created-rsa is
// replaced by the gate's own, more specific reason once the gate names one of the two reasons it
// is licensed to override with.
func TestRefineSchemeReasons_ReplacesNoTTYReason(t *testing.T) {
	schemes := fpscheme.ComputeAll(keyfile.Material{}) // nothing derivable at all
	derived := DerivedMaterial{Unlocked: false, Reason: domain.ReasonPassphraseRequiredNoTTY}

	refined := refineSchemeReasons(schemes, derived)
	for _, sf := range refined {
		if sf.Scheme != domain.SchemeAWSCreatedRSA {
			continue
		}
		if sf.Reason != domain.ReasonPassphraseRequiredNoTTY {
			t.Errorf("aws-created-rsa Reason = %q, want %q", sf.Reason, domain.ReasonPassphraseRequiredNoTTY)
		}
	}
}

// TestRefineSchemeReasons_NeverOverridesSchemeNotApplicable proves the stated exclusion: a
// derived.Reason of ReasonSchemeNotApplicable (T39 rule 2 — a key already known not to be RSA,
// never worth prompting for) must never overwrite anything, and in particular must not turn an
// honest "not applicable" into a misleading "no TTY."
func TestRefineSchemeReasons_NeverOverridesSchemeNotApplicable(t *testing.T) {
	schemes := fpscheme.ComputeAll(keyfile.Material{})
	before := make([]domain.SchemeFingerprint, len(schemes))
	copy(before, schemes)

	derived := DerivedMaterial{Unlocked: false, Reason: domain.ReasonSchemeNotApplicable}
	refined := refineSchemeReasons(schemes, derived)

	for i := range before {
		if refined[i] != before[i] {
			t.Errorf("scheme %s changed from %+v to %+v, want no change (ReasonSchemeNotApplicable must never override)", before[i].Scheme, before[i], refined[i])
		}
	}
}

// TestRefineSchemeReasons_NeverOverridesASuccessfulValue proves the second stated exclusion: an
// already-computed (derived) scheme is never touched, even when derived.Reason names one of the
// two overridable reasons.
func TestRefineSchemeReasons_NeverOverridesASuccessfulValue(t *testing.T) {
	_, m := openKnownMaterial(t, "rsa-pem-plain-pub") // unencrypted: aws-created-rsa computes successfully
	schemes := fpscheme.ComputeAll(m)

	derived := DerivedMaterial{Unlocked: false, Reason: domain.ReasonPassphraseRequiredNoTTY}
	refined := refineSchemeReasons(schemes, derived)

	for i, sf := range refined {
		if sf != schemes[i] {
			t.Errorf("scheme %s changed from %+v to %+v, want no change (a successfully computed value must never be overridden)", sf.Scheme, schemes[i], sf)
		}
	}
}

// TestKnownMaterialWithAgentFact_MergesAgentCommentBeforeReturning proves
// knownMaterialWithAgentFact itself performs the merge (investigateKey's step 3) before ever
// returning — the half of the ordering obligation that needs no PassphraseGate at all to observe.
func TestKnownMaterialWithAgentFact_MergesAgentCommentBeforeReturning(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	copyFixture(t, "ed25519-openssh-plain-pub", keyPath)
	copyFixture(t, "ed25519-openssh-plain-pub.pub", keyPath+".pub")

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}

	const wantComment = "agent-merged-comment"
	pub, err := keyfile.OpenMaterial(keyPath, nil)
	if err != nil || pub.Public == nil {
		t.Fatalf("OpenMaterial(%s, nil): mat=%+v err=%v", keyPath, pub, err)
	}
	facts := []sshagent.Fact{{Public: pub.Public, Comment: wantComment, Source: domain.FactSourceAgent}}

	_, known, agentComment, commentSource := knownMaterialWithAgentFact(m.Keys[0], facts)
	if commentSource != domain.FactSourceAgent {
		t.Fatalf("commentSource = %q, want %q", commentSource, domain.FactSourceAgent)
	}
	if agentComment != wantComment {
		t.Fatalf("agentComment = %q, want %q", agentComment, wantComment)
	}
	if known.Comment != wantComment {
		t.Fatalf("known.Comment = %q, want %q — the merge into known.Comment must have already happened by the time this function returns", known.Comment, wantComment)
	}
}

// TestInvestigateKey_AgentTriedBeforeDerive is this chunk's own load-bearing ordering obligation
// (tdd.md §18's ssh-agent subsection, roadmap.md §5.6's M3.6.4 chunk row): the agent lookup must
// run, and its result merge into a candidate key's material, BEFORE PassphraseGate.Derive is ever
// called for that key.
//
// **Rewritten in revision cycle 1 (verifier finding 3) because the prior version of this test did
// not actually guard that.** It asserted on knownMaterialWithAgentFact's own return values and on
// two hand-constructed PassphraseGate.Derive calls the test built itself — never on
// investigateKey's real output — while investigateKey's own AgentComment field was sourced
// directly from knownMaterialWithAgentFact's local, never from anything gate.Derive touched. That
// meant a call site with the ordering inverted (Derive called against a freshly re-opened,
// pre-merge Material) produced byte-identical investigateKey output to the correctly-ordered call,
// and this test passed either way — a guard that does not distinguish the two things it exists to
// distinguish is not a guard. With this chunk's finding-3 fix (investigateKey now reads AgentComment
// from derived.Material.Comment, guarded on commentSource == domain.FactSourceAgent) and finding-2
// fix (PassphraseGate.Derive now carries a caller-merged comment through its successful-unlock
// path too, T49) both landed, the ordering has an observable consequence at last, and this test
// pins it by asserting on investigateKey's real return value alone — no hand-built Derive calls.
//
// testdata/keys/ed25519-openssh-encrypted-nopub is the fixture the finding names: OpenSSH format
// (so known.Public is populated from the embedded public half, and CreatedRSAPromptWorthy reports
// false — never worth prompting, since ed25519 is never an aws-created-rsa candidate), genuinely
// encrypted, no `.pub` sidecar, and its own on-disk comment is empty (genfixtures used
// `ssh-keygen -C ""`). The *only* way a non-empty comment can reach investigateKey's output for
// this key is through the agent merge happening before Derive — there is no sidecar and no
// decrypt-recoverable comment (T49) to supply one any other way. Both PassphraseGate
// configurations below reach PassphraseGate.Derive's identical "not worth prompting" early return
// (Material: known verbatim, rule 2) for this specific key, since ed25519 never triggers a prompt
// regardless of whether a callback is present — which is exactly why both are exercised: proving
// the ordering guarantee holds for a gate with no passphrase source at all (the locked/no-TTY
// shape) and for one that would successfully unlock a *different* key if asked (the
// unlocked/correct-passphrase shape, Finding 2's own distinguishing case elsewhere in this
// package), not merely for one hand-picked configuration.
func TestInvestigateKey_AgentTriedBeforeDerive(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	copyFixture(t, "ed25519-openssh-encrypted-nopub", keyPath)

	m, err := Derive(DeriveOptions{KeyDir: dir})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(m.Keys) != 1 {
		t.Fatalf("len(m.Keys) = %d, want 1", len(m.Keys))
	}
	k := m.Keys[0]

	pub, err := keyfile.OpenMaterial(keyPath, nil)
	if err != nil || pub.Public == nil {
		t.Fatalf("OpenMaterial(%s, nil): mat=%+v err=%v", keyPath, pub, err)
	}
	const wantComment = "agent-tried-first-comment"
	facts := []sshagent.Fact{{Public: pub.Public, Comment: wantComment, Source: domain.FactSourceAgent}}

	cases := []struct {
		name string
		gate *PassphraseGate
	}{
		{"locked: no passphrase callback at all (T39's no-TTY shape)", &PassphraseGate{}},
		{"unlocked: a correct-passphrase callback is present", &PassphraseGate{Passphrase: func() ([]byte, error) {
			return []byte(testPassphrase), nil
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := investigateKey(k, InvestigateRequest{AgentFacts: facts, Gate: tc.gate})
			if got.CommentSource != domain.FactSourceAgent {
				t.Fatalf("CommentSource = %q, want %q", got.CommentSource, domain.FactSourceAgent)
			}
			if got.AgentComment != wantComment {
				t.Fatalf("AgentComment = %q, want %q — this is exactly the ordering guarantee this test exists to pin: it is only ever correct when the agent lookup and its merge into the candidate key's material ran before PassphraseGate.Derive was called for that key", got.AgentComment, wantComment)
			}
		})
	}
}
