package enum

type TaskState string

const (
	Todo       TaskState = "todo"
	InProgress TaskState = "in_progress"
	Done       TaskState = "done"
)

func (s TaskState) Valid() bool    { return s == Todo || s == InProgress || s == Done }
func (s TaskState) String() string { return string(s) }
