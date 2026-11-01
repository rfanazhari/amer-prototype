package enum

// Role is a fixed administrative access role; arbitrary converted strings must be validated at construction and restoration.
type Role string

const (
	Admin Role = "ADMIN"
	HR    Role = "HR"
)

// Valid reports whether the role is Admin or HR.
func (r Role) Valid() bool { return false }

// String returns the stable storage and API representation of the role.
func (r Role) String() string { return string(r) }
