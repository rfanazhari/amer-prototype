package enum

// AccountStatus is a fixed administrative account eligibility state; arbitrary converted strings must be validated at restoration.
type AccountStatus string

const (
	Active   AccountStatus = "ACTIVE"
	Disabled AccountStatus = "DISABLED"
)

// Valid reports whether the status is Active or Disabled.
func (s AccountStatus) Valid() bool { return s == Active || s == Disabled }

// String returns the stable storage and API representation of the status.
func (s AccountStatus) String() string { return string(s) }
