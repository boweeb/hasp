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

// The v1 finding id set (T29's table), fixed from the start even though M1's read-only scope
// means only some of these can ever actually fire yet (adopt-dependent repair paths don't exist,
// and stanza-in-multiple-groups/stanza-in-no-group are partial-apply artifacts of a write path
// M1 doesn't have — see check_host.go).
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
