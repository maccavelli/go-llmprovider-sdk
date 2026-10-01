package catalog

// Profile selects how the open catalogs rank their recommended models
// (MADR 0009 §1).
type Profile int

const (
	// ProfileUtility (the zero value) ranks for short, frequent tasks such as
	// commit messages: reasoning-capable, recent, paid, cheap.
	ProfileUtility Profile = iota
	// ProfileCapable ranks for reasoning-heavy tasks: strongest first.
	ProfileCapable
)

// ReasoningEffort is the recommended request effort for the profile: "low"
// for ProfileUtility, and "" for ProfileCapable, meaning each provider's
// documented default (see config.ReasoningEffort; MADR 0013 Q2). A
// value outside the two profiles is treated as ProfileUtility.
func (p Profile) ReasoningEffort() string {
	if p == ProfileCapable {
		return ""
	}
	return effortLow
}
