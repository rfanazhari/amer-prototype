// Package interviewrepository persists interview progress with GORM, independent of lifecycle and providers.
package interviewrepository

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"gorm.io/gorm"
)

// ErrUnmappable reports progress state that version 10 tables cannot represent faithfully.
// It is a technical error: the store refuses the write instead of dropping evidence.
var ErrUnmappable = errors.New("interview progress state is not representable in storage")

// Store persists interview progress through a shared GORM pool. Pins resolves the revision pin and
// lifecycle running instant that progress rows do not copy; Clock is the trusted application time.
type Store struct {
	DB    *gorm.DB
	Pins  interviewport.PinReader
	Clock service.Clock
}

var _ interviewport.ProgressStore = Store{}

// classify maps a PostgreSQL unique violation to the port's conflict; other failures stay technical.
func classify(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return interviewport.ErrConflict
	}
	return err
}

// instant reads trusted time at UTC millisecond precision.
func (s Store) instant() time.Time { return s.Clock.Now().UTC().Truncate(time.Millisecond) }
