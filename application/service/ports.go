package service

import (
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/event"
)

var ErrEventUnavailable = errors.New("event queue unavailable")

// Clock supplies trusted system time to application operations without coupling domain rules to wall time.
type Clock interface {
	// Now returns the trusted observation instant for an application operation.
	Now() time.Time
}

// Reservation holds event capacity until a transaction either commits and publishes or fails and cancels.
type Reservation interface {
	// Publish accepts a committed event exactly once using the reserved capacity.
	Publish(event.TaskCompleted)
	// Cancel releases unused capacity without publication and is idempotent.
	Cancel()
}

// EventQueue reserves publication capacity without publishing an uncommitted completion.
type EventQueue interface {
	// Reserve holds capacity or returns ErrEventUnavailable without publishing.
	Reserve() (Reservation, error)
}
