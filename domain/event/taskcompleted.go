package event

import "time"

type TaskCompleted struct {
	TaskID      string
	CompletedAt time.Time
}
