package domain

// Host is a destination and its settings — the content of SSH configuration (design.md §5.4).
type Host struct {
	Patterns  []string      `json:"patterns"`  // the "Host" line's arguments
	HostGroup string        `json:"hostGroup"` // absolute path of the host-group file this stanza lives in (D9)
	Managed   bool          `json:"managed"`   // inside hasp's markers? (D7, D14) — always false in M1
	Bindings  []Binding     `json:"bindings"`  // explicit and implicit-default, resolved and labelled (§5, T16)
	Profiles  []ProfilePath `json:"profiles"`  // derived from Bindings (T5)
}
