package domain

// This file is the frozen vocabulary `--investigate` (D21, tdd.md §18) reports beyond a plain
// read: Confidence (P10, D22, T37), the fingerprint-scheme identity space (T35, T36), and the
// origin-evidence shape (T37). It lives in internal/domain, stdlib-only (T13), because tdd.md §16
// row 7 makes the confidence vocabulary part of the frozen public compatibility surface, and a
// compatibility surface belongs with hasp's other core types rather than inside the adapter
// package (internal/adapter/fpscheme) that computes values for it. The scheme *registry* — the
// code that actually hashes bytes — lives in fpscheme instead, precisely because internal/domain
// may import nothing beyond the standard library (T13, layering_test.go) and computing a scheme
// needs crypto/x509, crypto/sha1, crypto/md5, and golang.org/x/crypto/ssh.

// Confidence is P10's closed, permanent status vocabulary (D22, T37): every fact --investigate
// reports beyond a plain read carries exactly one of these four values. tdd.md §16 row 7 makes
// this table part of the frozen compatibility surface — adding, removing, or redefining a value
// is a breaking (major-version) change, unlike Severity (internal/app/finding.go, T29), which
// T37 contrasts directly as deliberately re-tunable. AllConfidences is the golden-list guard
// test's (tdd.md §12) source of truth for the exact set.
type Confidence string

const (
	// ConfidenceDerived is a fact read directly from the artifact — what a plain read already
	// reports, and what a successfully computed fingerprint scheme value is (§5.1).
	ConfidenceDerived Confidence = "derived"
	// ConfidenceConfirmed is a fact established by matching external evidence the user
	// supplied — a console fingerprint matched under one of the two AWS RSA schemes, which is
	// deterministic proof of provenance (T36).
	ConfidenceConfirmed Confidence = "confirmed"
	// ConfidencePossible is a fact consistent with the evidence but not established — an
	// ED25519 origin guess, or a match under the SSH-native or legacy MD5 schemes, neither of
	// which carries AWS provenance signal on its own (T36).
	ConfidencePossible Confidence = "possible"
	// ConfidenceUnknown is a fact that cannot be determined. It is a first-class answer, never
	// a guess (§5.1, D12) — including, in this package, every registered scheme's honest
	// "could not compute" shape (see SchemeFingerprint below).
	ConfidenceUnknown Confidence = "unknown"
)

func (c Confidence) String() string { return string(c) }

func (c Confidence) MarshalJSON() ([]byte, error) { return marshalStringer(c) }

// AllConfidences returns the closed, four-value confidence set in the fixed order the golden-
// list guard test asserts against (tdd.md §12, mirroring T29's finding-id golden list). The
// order is documentation, not semantics — but a fifth entry, a dropped one, or a reordering that
// a careless refactor produces is exactly what naming this function's output "the golden list"
// and asserting it by name is meant to catch.
func AllConfidences() []Confidence {
	return []Confidence{ConfidenceDerived, ConfidenceConfirmed, ConfidencePossible, ConfidenceUnknown}
}

// SchemeID names one entry in the fingerprint scheme registry (internal/adapter/fpscheme, T35).
// The identity space lives here, in internal/domain, rather than in fpscheme itself, because
// SchemeFingerprint below — part of the frozen vocabulary --investigate reports — needs a typed
// id independent of the adapter package that computes values for it; fpscheme imports domain,
// never the reverse (T13).
//
// SchemeAWSCreatedRSA and SchemeAWSImportedRSA are named verbatim in tdd.md §18 and T36
// ("aws-created-rsa", "aws-imported-rsa") and must match those strings exactly. The other two
// ids are this chunk's own naming judgment call, since tdd.md's table names the schemes only in
// prose ("AWS ED25519 (created or imported)", "Legacy SSH MD5"): SchemeSSHNativeSHA256 and
// SchemeLegacySSHMD5.
type SchemeID string

const (
	SchemeAWSCreatedRSA   SchemeID = "aws-created-rsa"
	SchemeAWSImportedRSA  SchemeID = "aws-imported-rsa"
	SchemeSSHNativeSHA256 SchemeID = "ssh-native-sha256"
	SchemeLegacySSHMD5    SchemeID = "legacy-ssh-md5"
)

func (s SchemeID) String() string { return string(s) }

func (s SchemeID) MarshalJSON() ([]byte, error) { return marshalStringer(s) }

// ReasonToken is a machine-readable evidence token for SchemeFingerprint.Reason and
// Origin.Because (T37): either a short kebab phrase (a fixed vocabulary, the constants below) or
// a dynamically formatted "key=value" pair (tdd.md §10's own worked examples: "algorithm=rsa",
// "format=pem") — never prose. Free text belongs to the human renderer alone, which is free to
// turn the same tokens into a sentence; nothing that reads --investigate's --json output
// mechanically should ever need to parse English out of these fields.
//
// Unlike Confidence, this set is not declared a frozen compatibility surface by tdd.md §16 — a
// reason is descriptive evidence, closer in spirit to a check finding's re-tunable Severity (T29)
// than to Confidence's closed, safety-gated vocabulary. New tokens are added as new derivation
// paths are added; existing ones are not repurposed to mean something else, the same courtesy
// every other stable identifier in this codebase gets even where nothing forces it.
type ReasonToken string

const (
	// ReasonSchemeNotApplicable marks a scheme that does not apply to this key's algorithm at
	// all — the two AWS RSA schemes (T35) against a non-RSA key, for instance. A scheme this
	// reason applies to is never silently omitted from --investigate's output (roadmap.md §5.6
	// exit criterion 2); it is reported with this reason instead.
	ReasonSchemeNotApplicable ReasonToken = "scheme-not-applicable"
	// ReasonKeyEncodingUnsupported marks a key whose encoding fails Go's own marshaling for a
	// scheme that otherwise applies to it — T35's own example is a DSA public key failing
	// x509.MarshalPKIXPublicKey: "that too is an honest unknown with a reason, not a panic and
	// not an omission."
	ReasonKeyEncodingUnsupported ReasonToken = "key-encoding-unsupported"
	// ReasonPublicHalfUnavailable marks a scheme that needs only the public half (T35's
	// declared-material distinction) when even that could not be derived — §5.1's derivation
	// gap propagating into a scheme's own Reason instead of being silently absorbed.
	ReasonPublicHalfUnavailable ReasonToken = "public-half-unavailable"
	// ReasonPrivateKeyUnavailable marks a scheme that needs the decrypted private key (T35)
	// when hasp was not given one for this key this invocation — either no passphrase was
	// supplied at all, or (chunk M3.6.2) neither the agent nor a prompt produced one.
	ReasonPrivateKeyUnavailable ReasonToken = "private-key-unavailable"
	// ReasonPassphraseRequiredNoTTY is the literal token roadmap.md §5.6 exit criterion 5
	// mandates: with no TTY attached, --investigate degrades rather than prompting (T39, D19),
	// and everything that stayed unknown for exactly that reason carries this token. Defined
	// here, ahead of the passphrase-prompting caller M3.6.2 adds, because this file is this
	// vocabulary's one documented home (tdd.md §18) and the token's exact spelling is a
	// contract chunk M3.6.2 must land on, not something it should be free to improvise.
	ReasonPassphraseRequiredNoTTY ReasonToken = "passphrase-required-no-tty"
	// ReasonNoConsoleFingerprintSupplied is tdd.md §10's own worked example: an origin guess
	// made with no external console fingerprint available to confirm it against.
	ReasonNoConsoleFingerprintSupplied ReasonToken = "no-console-fingerprint-supplied"
)

// SchemeFingerprint is one registered scheme's answer for one key. roadmap.md §5.6 exit
// criterion 2 requires every registered scheme to appear in --investigate's output — a scheme
// that could not be computed reports the "could not compute" shape (Value == "", Confidence ==
// ConfidenceUnknown, Reason non-empty) rather than being omitted; no scheme is ever silently
// dropped from the response.
type SchemeFingerprint struct {
	// Scheme carries the literal JSON key "scheme", deliberately not "id" even though Origin.ID
	// below is exactly the same shape of thing — a stable identifier for what produced this
	// value. The asymmetry is intentional, not an inconsistency a future reader should "tidy":
	// the two are different namespaces (a fingerprint scheme's own id, SchemeID — "aws-created-
	// rsa" and friends, T35 — versus an inferred origin's id, a plain string — "aws-ec2-created"
	// and friends, T37), and naming both fields "id" would blur that distinction on the wire for
	// no reason beyond surface symmetry. Ratified before the v1.0.0 freeze (tdd.md §16 row 2 makes
	// the envelope shape, and by extension each kind's `data` field names, part of the frozen
	// compatibility surface); left as-is by deliberate maintainer decision during M3.6.3, not an
	// oversight carried forward.
	Scheme     SchemeID    `json:"scheme"`
	Value      string      `json:"value"`
	Confidence Confidence  `json:"confidence"`
	Reason     ReasonToken `json:"reason,omitempty"`
}

// Origin is one provenance guess about a key — "this key was created directly on AWS EC2," for
// instance (T37, T36). JSON tags match tdd.md §10's and T37's literal worked example exactly:
// {"id": "aws-ec2-created", "confidence": "possible", "because": [...]}.
//
// Origin ids are a namespace separate from SchemeID: which scheme a clue matched under is not
// the same claim as which origin hasp infers from it (T36's whole point — a match under
// SchemeSSHNativeSHA256 or SchemeLegacySSHMD5 proves nothing about provenance, only a match
// under SchemeAWSCreatedRSA or SchemeAWSImportedRSA does). OriginAWSEC2Created and
// OriginAWSEC2Imported are T37's own worked-example names.
type Origin struct {
	ID         string        `json:"id"`
	Confidence Confidence    `json:"confidence"`
	Because    []ReasonToken `json:"because"`
}

const (
	OriginAWSEC2Created  = "aws-ec2-created"
	OriginAWSEC2Imported = "aws-ec2-imported"
)

// FactSource labels *where* --investigate obtained a fact, when that provenance itself matters
// enough to travel with the value rather than be inferred from context (T38, chunk M3.6.2). It
// is a narrower, deliberately separate concept from Confidence (P10) above: Confidence says how
// sure hasp is that a value is right; FactSource says where hasp got it from. The two compose —
// an agent-sourced comment is reported at ConfidenceDerived (it is a plain fact, not a guess)
// and FactSourceAgent (it is real only for as long as the agent that supplied it keeps running).
//
// Forward note, chunk M3.6.4: once FactSource actually reaches --json (it does not yet — this
// vocabulary is defined here, ahead of its first --investigate caller, for the reason stated
// below), it becomes a consumer-filterable safety signal in exactly the sense Confidence already
// is — a consumer deciding whether to trust an agent-sourced, only-real-while-the-agent-runs fact
// is making the same kind of decision T37 describes for Confidence. It should get the same
// closed-set golden-list guard test (tdd.md §12) Confidence already has then, not be left to a
// reviewer's eye the way a smaller, two-value set might tempt someone to skip.
//
// FactSource is intentionally not folded into Confidence's closed four-value set: doing so would
// either invent a fifth confidence value (forbidden, T37's frozen compatibility-surface row) or
// silently misreport an agent-sourced fact as an ordinary derived one — the exact
// labelled-never-merged treatment T16 already established for Binding.Kind
// (BindingExplicit vs. BindingImplicitDefault), which T38 explicitly extends to this case:
// "labelled agent-sourced, never merged into the plain derived bucket."
//
// It lives here, alongside Confidence, rather than as a bare string field on
// internal/adapter/sshagent.Fact alone, so a future consumer has one typed, documented
// vocabulary to check regardless of which source produced the fact — sshagent.Fact also carries
// it directly (as this exact type), so the label travels with the value from the moment it is
// produced rather than needing to be reattached downstream.
type FactSource string

const (
	// FactSourcePlainRead is a plain read's own derivation — the default for everything §5-§11
	// already report. It needs no visible label in practice, since there is only one source for
	// it; named here only so FactSourceAgent below has an explicit zero-value counterpart rather
	// than relying on Go's implicit "" for a claim this file otherwise states outright.
	FactSourcePlainRead FactSource = ""
	// FactSourceAgent labels a fact internal/adapter/sshagent supplied (T38): real only for as
	// long as the agent that reported it keeps running, which is why it is never merged into the
	// plain derived bucket above.
	FactSourceAgent FactSource = "agent-sourced"
)
