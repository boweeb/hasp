package app

import (
	"github.com/boweeb/hasp/internal/adapter/fpscheme"
	"github.com/boweeb/hasp/internal/adapter/keyfile"
	"github.com/boweeb/hasp/internal/adapter/sshagent"
	"github.com/boweeb/hasp/internal/domain"
)

// InvestigatedKey is --investigate's per-key projection (D21, tdd.md §18) — additive over a plain
// read's domain.Key, never replacing it. Embedding domain.Key rather than duplicating its fields
// is deliberate and load-bearing for roadmap.md §5.6's frozen v1.0.0 compatibility surface
// (tdd.md §16): Key carries no custom MarshalJSON of its own (only its field types do — KeyFormat,
// KeyIdentity's byFingerprint/byPath), so embedding marshals every existing field flat, at its
// existing JSON path, unchanged — an existing consumer reading key.list/key.show's `kind` and
// field shape sees nothing new unless it also reads one of the fields below. `kind` itself stays
// "key.list"/"key.show" (internal/cli/key.go) — no new kind string, exactly as tdd.md §16 row 3
// requires.
type InvestigatedKey struct {
	domain.Key
	// Schemes is every registered fingerprint scheme's answer for this key (T35), in registry
	// order (fpscheme.Schemes), never fewer — roadmap.md §5.6 exit criterion 2.
	Schemes []domain.SchemeFingerprint `json:"schemes"`
	// Origins is T37's provenance-guess evidence, computed with no external clue supplied — see
	// originsForInvestigate's own doc comment for the RSA/non-RSA rule.
	Origins []domain.Origin `json:"origins"`
	// AgentComment and CommentSource are T38's agent-sourced fact: a loaded key's comment,
	// cross-referenced by public key (internal/adapter/sshagent.MatchComment), labelled rather
	// than merged into Key.Comment. Both stay zero-valued (and omitted from --json) when no
	// agent match was found — CommentSource's zero value is domain.FactSourcePlainRead, which
	// carries no useful meaning here since Key.Comment, not this field, is the plain-read fact;
	// a consumer's signal for "was there an agent match" is CommentSource ==
	// domain.FactSourceAgent, never an empty-string check on AgentComment alone (an agent can
	// legitimately report an empty comment).
	AgentComment  string            `json:"agentComment,omitempty"`
	CommentSource domain.FactSource `json:"commentSource,omitempty"`
}

// InvestigatedKeyDetail is show key --investigate's full-detail answer — the same parallel shape
// KeyDetail already establishes for a plain read (Key + Hosts), with Key's type swapped for
// InvestigatedKey. The `key`/`hosts` JSON paths are unchanged from KeyDetail's own (tdd.md §16);
// only what lives under `key` grows.
type InvestigatedKeyDetail struct {
	Key   InvestigatedKey `json:"key"`
	Hosts []domain.Host   `json:"hosts"`
}

// InvestigateRequest carries every input --investigate's use case needs, gathered by
// internal/cli before this package is ever called — internal/app never reads an env var, a TTY,
// or a socket path itself (tdd.md §3's Request-only framing, already established for
// PassphraseGate and extended here to the agent side of the same flag).
type InvestigateRequest struct {
	// AgentFacts is internal/cli's call to sshagent.List(os.Getenv("SSH_AUTH_SOCK")) — already
	// fail-open by that function's own contract (nil on no agent, dead socket, or protocol
	// error), so a nil slice here is simply "nothing to cross-reference," never a signal this
	// package needs to special-case.
	AgentFacts []sshagent.Fact
	// Gate is D19/T39's passphrase policy for this invocation, or nil when the caller has none
	// to offer (internal/cli always supplies one; nil is accepted so this package's own tests,
	// and any future caller, are not forced to construct one just to get a no-op).
	Gate *PassphraseGate
}

// InvestigateKeys is list key --investigate's use case: every key in m, each run through
// investigateKey below.
func InvestigateKeys(m Machine, req InvestigateRequest) []InvestigatedKey {
	out := make([]InvestigatedKey, 0, len(m.Keys))
	for _, k := range m.Keys {
		out = append(out, investigateKey(k, req))
	}
	return out
}

// InvestigateKey is show key --investigate's use case, scoped to one key (tdd.md §9's show-key
// cell: "the same surface as list key --investigate, scoped to one key"). It reuses ShowKey's own
// lookup rather than duplicating it, so the two commands can never disagree about which key a
// name resolves to or which hosts bind it.
func InvestigateKey(m Machine, name string, req InvestigateRequest) (InvestigatedKeyDetail, bool) {
	detail, ok := ShowKey(m, name)
	if !ok {
		return InvestigatedKeyDetail{}, false
	}
	return InvestigatedKeyDetail{Key: investigateKey(detail.Key, req), Hosts: detail.Hosts}, true
}

// investigateKey assembles one key's --investigate projection, and is the one call site in this
// tree where tdd.md §18's `ssh-agent` subsection's forward obligation, restated verbatim in
// roadmap.md §5.6's M3.6.4 chunk row, is actually discharged: "the agent lookup runs, and its
// result merges into a candidate key's material, BEFORE PassphraseGate.Derive is ever called for
// that key." Neither internal/adapter/sshagent nor PassphraseGate can enforce that ordering on
// its own — sshagent.MatchComment and PassphraseGate.Derive are both pure with respect to each
// other, and only a caller invoking both, in order, makes the ordering real. The five numbered
// steps below are that ordering, made legible on sight rather than merely asserted in this
// comment, and are never to be reordered.
//
// **The ordering is pinned by an observable consequence, not by decomposition alone** (revision
// cycle 1, verifier finding 3). The five-step split below was already structurally correct — step
// 4 cannot compile without step 3's `known` already in hand — but a call site that quietly
// re-derived a *second*, pre-merge Material and handed that to Derive instead would still
// compile, and the output this function returns would not have changed at all before this
// revision: AgentComment was sourced directly from knownMaterialWithAgentFact's own return value,
// never from derived.Material.Comment, so the merge's presence in `known` had no observable effect
// downstream. AgentComment is now sourced from derived.Material.Comment whenever commentSource
// names the agent as the source — which only reads correctly, and only ever carries the agent's
// comment, when knownMaterialWithAgentFact's merge actually happened before gate.Derive was called
// with that merged value (PassphraseGate.Derive's own doc comment, T49's carry-forward fix). A
// call site that inverted the ordering now produces a visibly wrong AgentComment (empty, or a
// stale pre-merge value) rather than a silently-still-correct one — see
// TestInvestigateKey_AgentTriedBeforeDerive's own doc comment for the inverted-ordering proof this
// property makes possible.
func investigateKey(k domain.Key, req InvestigateRequest) InvestigatedKey {
	gate := req.Gate
	if gate == nil {
		// A caller supplying no gate at all gets the same honest degrade a gate with no
		// Passphrase callback already produces (T39's "nobody to ask" case) — never a nil-
		// pointer panic reaching into internal/cli's own construction discipline.
		gate = &PassphraseGate{}
	}

	// Steps 1-3, split into their own function (knownMaterialWithAgentFact below) precisely so
	// the ordering this function's doc comment names is a structural property of the code, not
	// merely a comment a future edit could silently invalidate: every call path through this
	// function computes known, agentComment, and commentSource — with the agent merge already
	// applied to known — before step 4 (gate.Derive) ever runs, because that call cannot compile
	// without path and known already in hand from this one, unconditionally-first call.
	// The third return value (this function's own local agent-sourced comment, prior to
	// revision cycle 1) is intentionally discarded here with `_`: AgentComment is now sourced
	// from derived.Material.Comment below, once step 4 has run, rather than from this earlier
	// local — see this function's own doc comment for why that switch is what makes the
	// ordering guarantee observable rather than merely structural. commentSource remains the
	// authoritative "was there an agent match" signal (InvestigatedKey's own doc comment) and
	// is unaffected by this change.
	path, known, _, commentSource := knownMaterialWithAgentFact(k, req.AgentFacts)

	// 4. Only now does a passphrase ever get asked for, and only for this one key — path == ""
	// (no location to open at all) skips the gate entirely rather than asking Derive to open an
	// empty path: D19/T39 license a prompt only for an operation that could actually use the
	// answer, and an empty path can never produce one.
	var derived DerivedMaterial
	if path == "" {
		derived = DerivedMaterial{Material: known, Unlocked: false, Reason: domain.ReasonPrivateKeyUnavailable}
	} else {
		derived = gate.Derive(path, known)
	}

	// 5. Every registered scheme, computed from whatever material step 4 produced.
	schemes := fpscheme.ComputeAll(derived.Material)
	schemes = refineSchemeReasons(schemes, derived)

	// AgentComment is read from derived.Material.Comment — the material that actually flowed
	// through gate.Derive — rather than from knownMaterialWithAgentFact's own local, precisely so
	// a call site that reordered steps 3 and 4 produces an observably wrong answer (see this
	// function's own doc comment above, and the revision-cycle-1 note on why that matters).
	// Guarded on commentSource == domain.FactSourceAgent, never read unconditionally:
	// keyfile.OpenMaterial populates Material.Comment from an ordinary `.pub` sidecar too, and
	// reading it here with no guard would mislabel that plain-read fact as agent-sourced — exactly
	// the "labelled, never merged into the plain derived bucket" rule T38 and domain.FactSource's
	// own doc comment require.
	reportedAgentComment := ""
	if commentSource == domain.FactSourceAgent {
		reportedAgentComment = derived.Material.Comment
	}

	return InvestigatedKey{
		Key:           k,
		Schemes:       schemes,
		Origins:       originsForInvestigate(k),
		AgentComment:  reportedAgentComment,
		CommentSource: commentSource,
	}
}

// knownMaterialWithAgentFact performs investigateKey's steps 1-3: open the public half with no
// passphrase (step 2), then cross-reference facts by public key and merge any match into the
// returned Material's own Comment field (step 3) — all of it before a passphrase is ever asked
// for. Split out from investigateKey so the merge itself is directly testable in isolation
// (TestKnownMaterialWithAgentFact_MergesAgentCommentBeforeReturning) and so the ordering
// obligation is provable at the call-sequence level (TestInvestigateKey_AgentTriedBeforeDerive):
// keyfile.Material is a plain value type, so a caller that calls PassphraseGate.Derive with a
// Material obtained *before* this function runs — rather than this function's own return value —
// hands Derive a stale copy that never saw the agent's comment at all, which is exactly the
// failure this split makes observable rather than merely asserted in prose.
//
// The agentComment return value is this function's own record of the match, used only by its own
// direct test above; investigateKey (revision cycle 1, finding 3) deliberately discards it with
// `_` and instead reads the reported comment back off derived.Material.Comment once
// PassphraseGate.Derive has run, so that the *actual* material Derive was called with — not merely
// this function's own bookkeeping — is what the output is built from. commentSource is still
// consumed directly from here: it is the authoritative "was there an agent match at all" signal
// (InvestigatedKey's own doc comment), independent of whether Derive's output still carries the
// comment string itself.
func knownMaterialWithAgentFact(k domain.Key, facts []sshagent.Fact) (path string, known keyfile.Material, agentComment string, commentSource domain.FactSource) {
	// 1. The one file, if any, this key's material would come from.
	path = primaryKeyPath(k)

	// 2. Public half only, no passphrase — never a private-key read yet (D19/P3's consent gate
	// has not fired at all at this point).
	if path != "" {
		if mat, err := keyfile.OpenMaterial(path, nil); err == nil {
			known = mat
		}
		// A read error here degrades the same way §11's fail-open row for an unreadable key
		// file already does elsewhere in the pipeline: known stays its zero value, and every
		// scheme investigateKey computes downstream reports its own honest "could not compute"
		// shape rather than this function aborting the whole investigation over one bad file.
	}

	// 3. Agent cross-reference. A match is recorded for output (agentComment/commentSource) AND
	// merged into known.Comment, which is the literal discharge of PassphraseGate.Derive's own
	// doc comment: "a caller doing this gets the fact preserved for free."
	if comment, ok := sshagent.MatchComment(facts, known.Public); ok {
		agentComment = comment
		commentSource = domain.FactSourceAgent
		known.Comment = comment
	}

	return path, known, agentComment, commentSource
}

// refineSchemeReasons implements roadmap.md §5.6 exit criterion 5's precision requirement:
// fpscheme.ComputeAll's own "could not compute" shape for a RequiresDecryptedPrivateKey scheme
// (aws-created-rsa, the only one) always reports domain.ReasonPrivateKeyUnavailable when
// Material.Private is nil, because ComputeAll has no visibility into *why* it is nil — it only
// ever sees the material, never the PassphraseGate decision that produced it. That is true but not
// the reason criterion 5 actually asks for ("marks the rest unknown with
// passphrase-required-no-tty"), so this function replaces it with derived.Reason whenever derived
// itself names a more precise cause: no TTY / no callback (domain.ReasonPassphraseRequiredNoTTY),
// or the gate genuinely could not produce a usable key (domain.ReasonPrivateKeyUnavailable, e.g. a
// wrong passphrase or an unreadable path — see PassphraseGate.Derive).
//
// Never overrides domain.ReasonSchemeNotApplicable — a non-RSA key genuinely does not apply to
// aws-created-rsa, and that is a different, already-correct claim from "the private key was
// unavailable." Never overrides a successfully computed value either: this function only ever
// touches an entry whose own Reason is exactly ReasonPrivateKeyUnavailable, which by
// fpscheme.computeOne's construction only occurs on the "could not compute" shape (Value == "",
// Confidence == unknown) in the first place — a scheme that succeeded never reaches this branch.
func refineSchemeReasons(schemes []domain.SchemeFingerprint, derived DerivedMaterial) []domain.SchemeFingerprint {
	if derived.Unlocked {
		return schemes
	}
	if derived.Reason != domain.ReasonPassphraseRequiredNoTTY && derived.Reason != domain.ReasonPrivateKeyUnavailable {
		return schemes
	}
	requiresPrivate := make(map[domain.SchemeID]bool, len(fpscheme.Schemes()))
	for _, s := range fpscheme.Schemes() {
		if s.Requires == fpscheme.RequiresDecryptedPrivateKey {
			requiresPrivate[s.ID] = true
		}
	}
	for i := range schemes {
		sf := &schemes[i]
		if requiresPrivate[sf.Scheme] && sf.Reason == domain.ReasonPrivateKeyUnavailable {
			sf.Reason = derived.Reason
		}
	}
	return schemes
}

// originsForInvestigate is --investigate's origin-evidence rule when no external clue is supplied
// — a genuinely different question from find key's originForSchemeMatch (internal/app/key.go),
// which grades a *matched* clue. --investigate has no clue to match at all, so it can never report
// domain.ConfidenceConfirmed (T36/T37: confirmed requires matching external evidence the user
// supplied) — the strongest thing it can ever say is possible.
//
// An RSA key yields BOTH domain.OriginAWSEC2Created and domain.OriginAWSEC2Imported, each at
// ConfidencePossible: both are genuinely consistent with an RSA key's evidence (T35's table — an
// AWS-created key is PEM-encoded PKCS#8/PKCS1, and an imported key can be PEM too), and reporting
// only one would imply the other is inconsistent, which is false. The because tokens carry the
// discriminating evidence (the key's own algorithm and on-disk format, plus the fixed reason
// domain.ReasonNoConsoleFingerprintSupplied) for a consumer to weigh — the verdict itself does not
// pretend to more than that.
//
// A non-RSA key yields an EMPTY origins array: neither AWS RSA scheme applies to it at all
// (fpscheme.IsRSAAlgorithm, the identical check importedRSAApplicable makes against a key's public
// half, so the two can never disagree), so there is no provenance signal to grade — inventing one
// would be exactly the guess design.md §5.1 / D12 forbid. (T36's own note: AWS's ED25519
// fingerprint is identical whether created or imported, so even a scheme match there proves
// nothing about provenance — --investigate, with no match at all to lean on, has even less.)
//
// This function returns a bare Go `nil` for that empty case — idiomatic Go, and, as of T52, also
// correct on the wire: InvestigatedKey.Origins is tagged `json:"origins"` with no `omitempty`, so
// it is always present in `hasp list key --investigate --json`'s output, but
// internal/cli/render/json.go's marshalData now normalizes every nil slice it can reach, at any
// struct-field depth, not only the top-level `data` value passed to render.JSON — the same
// guarantee that also now covers domain.Key.Profiles ("profiles") and app.KeyDetail.Hosts
// ("hosts"), both of which used to reach the wire as JSON `null` for exactly this reason before
// T52. One mechanism in the renderer, applied uniformly, is more structurally consistent than a
// hand-maintained `[]domain.Origin{}` special case at this one call site — the earlier version of
// this comment argued at length for keeping that special case specifically *because* marshalData
// was not yet recursive; T52 made it recursive, so the argument, and the special case it defended,
// are both retired. Nothing about the wire output changes: a non-RSA key's `origins` field still
// marshals as `[]`, never `null` — proven where that guarantee now actually lives, one layer up,
// by internal/cli/render's own unit tests and by internal/cli's end-to-end guard
// (TestJSONKinds_NeverEmitArrayTypedNull, internal/cli/jsonnullguard_test.go), which asserts it
// through the full --json pipeline for this kind's own `origins` field specifically, and for every
// other --json kind's own genuinely nested array fields the same way — T53
// (docs/tech-decision-log.md) records, as a corrected and deliberate fact rather than an
// oversight, that two of the guard's twelve enumerated kinds (profile.list, profile.find) are
// structural no-ops for this regression class: their payload types carry no nested array field at
// all, so neither can fail for this class of defect regardless of what normalizeNilSlices does. An
// earlier version of this comment cited a TestOriginsForInvestigate_JSON_NeverNull that T52 itself
// deleted — it tested the retired workaround via a bare json.Marshal that bypassed the renderer
// entirely, so it could not have proven the claim it was cited for even had it survived.
//
// tdd.md §10's / T37's worked example — {"id": "aws-ec2-created", "confidence": "possible",
// "because": ["algorithm=rsa", "format=pem", "no-console-fingerprint-supplied"]} — shows the shape
// of exactly one element of the two-element array this function returns for an RSA key; that shape
// stays exactly right here.
func originsForInvestigate(k domain.Key) []domain.Origin {
	if !fpscheme.IsRSAAlgorithm(k.Algorithm) {
		return nil
	}
	because := []domain.ReasonToken{
		domain.ReasonToken("algorithm=" + k.Algorithm),
		domain.ReasonToken("format=" + k.Format.String()),
		domain.ReasonNoConsoleFingerprintSupplied,
	}
	return []domain.Origin{
		{ID: domain.OriginAWSEC2Created, Confidence: domain.ConfidencePossible, Because: because},
		{ID: domain.OriginAWSEC2Imported, Confidence: domain.ConfidencePossible, Because: because},
	}
}
