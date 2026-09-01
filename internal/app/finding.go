package app

import (
	"encoding/json"
	"sort"
)

// Severity is advice, and may be re-tuned — it never affects check's exit code (T29, tdd.md
// §6.2: check is advisory and its findings are non-suppressible, so a severity gating the exit
// code would be a suppression mechanism arriving through the back door).
type Severity int

const (
	SeverityError   Severity = iota // something is actually broken
	SeverityWarning                 // it works but is probably not what was meant
	SeverityInfo                    // worth knowing, possibly deliberate
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return "unknown"
	}
}

func (s Severity) MarshalJSON() ([]byte, error) { return json.Marshal(s.String()) }

// FindingID is permanent: kebab-case, never renamed, never reused for a different meaning — the
// same guarantee a Dn or Tn id carries (T29). A finding that stops being reported keeps its id
// retired rather than recycled.
type FindingID string

// The v1 finding id set (T29's table), fixed from the start (T29: "the finding schema fixed from
// the start"). check itself never repairs anything — it stays advisory everywhere (tdd.md §9's
// grid: "repairing a finding it raises on an unmanaged resource requires adopt first") — but as of
// M2, every key- and profile-scoped finding below now has a real, in-hasp write verb a user can run
// in response to it: FindingKeyNoProfile and FindingUnmarkedProfileDir are addressed by `adopt
// key`/`adopt profile`; FindingEmptyProfileDir by `release profile`. FindingDuplicateKeyConfirmed,
// FindingDuplicateKeyUnconfirmed, FindingKeyMissingPublicHalf, and FindingFingerprintUnknown remain
// unrepairable by design, not by omission — no verb removes a redundant copy, regenerates a missing
// .pub, or resolves an undecidable fingerprint (D12 is permanent). Every host-scoped finding below
// (FindingDanglingIdentityFile, FindingUnresolvableToken, FindingRelativeIdentityFile,
// FindingShadowedStanza, FindingHostNoBinding, FindingMarkerDefect) is still report-only: `adopt
// host`/`edit host`/`release host` and every WriteRegion change are M3's own work, explicitly out
// of M2's scope (roadmap.md §4). FindingStanzaInMultipleGroups/FindingStanzaInNoGroup remain the
// reserved, always-empty detectors check_host.go's own doc comment describes — their real trigger
// (a stanza left in two host-group files, or none, by a partially-applied multi-file Plan) still
// needs WriteRegion and the metadata channel (T25), neither of which M2 builds either.
const (
	FindingDuplicateKeyConfirmed   FindingID = "duplicate-key-confirmed"
	FindingDuplicateKeyUnconfirmed FindingID = "duplicate-key-unconfirmed"
	FindingKeyMissingPublicHalf    FindingID = "key-missing-public-half"
	FindingKeyNoProfile            FindingID = "key-no-profile"
	FindingFingerprintUnknown      FindingID = "fingerprint-unknown"
	FindingDanglingIdentityFile    FindingID = "dangling-identityfile"
	FindingUnresolvableToken       FindingID = "unresolvable-token"
	FindingRelativeIdentityFile    FindingID = "relative-identityfile"
	FindingShadowedStanza          FindingID = "shadowed-stanza"
	FindingHostNoBinding           FindingID = "host-no-binding"
	FindingStanzaInMultipleGroups  FindingID = "stanza-in-multiple-groups"
	FindingStanzaInNoGroup         FindingID = "stanza-in-no-group"
	FindingEmptyProfileDir         FindingID = "empty-profile-dir"
	FindingUnmarkedProfileDir      FindingID = "unmarked-profile-dir"
	FindingMarkerDefect            FindingID = "marker-defect"
)

// findingSeverity is the permanent id -> re-tunable severity mapping.
var findingSeverity = map[FindingID]Severity{
	FindingDuplicateKeyConfirmed:   SeverityWarning,
	FindingDuplicateKeyUnconfirmed: SeverityInfo,
	FindingKeyMissingPublicHalf:    SeverityWarning,
	FindingKeyNoProfile:            SeverityInfo,
	FindingFingerprintUnknown:      SeverityInfo,
	FindingDanglingIdentityFile:    SeverityError,
	FindingUnresolvableToken:       SeverityWarning,
	FindingRelativeIdentityFile:    SeverityWarning,
	FindingShadowedStanza:          SeverityWarning,
	FindingHostNoBinding:           SeverityWarning,
	FindingStanzaInMultipleGroups:  SeverityError,
	FindingStanzaInNoGroup:         SeverityError,
	FindingEmptyProfileDir:         SeverityInfo,
	FindingUnmarkedProfileDir:      SeverityInfo,
	FindingMarkerDefect:            SeverityError,
}

// AllFindingIDs returns the fixed v1 finding id set, sorted, for the golden-list guard test
// (Stage 22): a new finding id then shows up in a diff rather than in a consumer's jq filter
// silently failing to match (T29).
func AllFindingIDs() []FindingID {
	ids := make([]FindingID, 0, len(findingSeverity))
	for id := range findingSeverity {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// Subject identifies what a Finding is about.
type Subject struct {
	Kind string `json:"kind"` // "key", "host", "profile", "host-group"
	Name string `json:"name"`
}

// Finding is one check result (T29).
type Finding struct {
	ID       FindingID      `json:"id"`
	Severity Severity       `json:"severity"`
	Subject  Subject        `json:"subject"`
	Message  string         `json:"message"`
	Detail   map[string]any `json:"detail,omitempty"`
}

func newFinding(id FindingID, subject Subject, message string, detail map[string]any) Finding {
	return Finding{ID: id, Severity: findingSeverity[id], Subject: subject, Message: message, Detail: detail}
}
