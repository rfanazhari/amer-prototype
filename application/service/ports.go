package service

import (
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/event"
)

var ErrEventUnavailable = errors.New("event queue unavailable")

type Clock interface{ Now() time.Time }
type Reservation interface {
	Publish(event.TaskCompleted)
	Cancel()
}
type EventQueue interface{ Reserve() (Reservation, error) }
