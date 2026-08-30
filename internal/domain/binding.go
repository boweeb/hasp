package domain

// BindingKind distinguishes an explicit IdentityFile directive from ssh's own default-identity
// probing (T16). The two are labelled, never merged into one undifferentiated set: "you wrote
// this" and "ssh would fall back to this" are different facts about the machine (P1).
type BindingKind int

const (
	BindingExplicit        BindingKind = iota // from an IdentityFile line
	BindingImplicitDefault                    // one of ssh_config(5)'s default identity paths (§5, T16)
)

func (k BindingKind) String() string {
	switch k {
	case BindingExplicit:
		return "explicit"
	case BindingImplicitDefault:
		return "implicit-default"
	default:
		return "unknown"
	}
}

func (k BindingKind) MarshalJSON() ([]byte, error) { return marshalStringer(k) }

// Binding is the relationship "this host is reached with this key" (design.md §5.6), resolved
// and labelled.
type Binding struct {
	Key  KeyIdentity `json:"key"`
	Kind BindingKind `json:"kind"`
}
