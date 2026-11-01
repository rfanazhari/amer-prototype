package enum

// FindingSeverity is a validation-finding class; errors block publication and warnings do not.
type FindingSeverity string

const (
	FindingError   FindingSeverity = "error"
	FindingWarning FindingSeverity = "warning"
)

// Valid reports whether the severity is an error or a warning.
func (s FindingSeverity) Valid() bool { return s == FindingError || s == FindingWarning }

// String returns the stable finding-payload representation of the severity.
func (s FindingSeverity) String() string { return string(s) }
