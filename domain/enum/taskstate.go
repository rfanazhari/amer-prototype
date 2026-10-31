package enum

// TaskState names the supported lifecycle states; arbitrary converted strings must be validated at restoration boundaries.
type TaskState string

const (
	Todo       TaskState = "todo"
	InProgress TaskState = "in_progress"
	Done       TaskState = "done"
)

// Valid reports whether the state is Todo, InProgress or Done.
func (s TaskState) Valid() bool { return s == Todo || s == InProgress || s == Done }

// String returns the stable storage and API representation of the state.
func (s TaskState) String() string { return string(s) }
