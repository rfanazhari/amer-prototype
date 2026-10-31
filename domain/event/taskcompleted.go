package event

import "time"

// TaskCompleted records the identity and system time of a successful domain completion. Producing this fact does not itself publish it.
type TaskCompleted struct {
	TaskID      string
	CompletedAt time.Time
}
