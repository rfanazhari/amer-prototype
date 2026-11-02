// Package interviewrepository persists interview progress with GORM, independent of lifecycle and providers.
package interviewrepository

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"gorm.io/gorm"
)

// ErrUnmappable reports progress state that version 10 tables cannot represent faithfully.
// It is a technical error: the store refuses the write instead of dropping evidence.
var ErrUnmappable = errors.New("interview progress state is not representable in storage")

// errNotImplemented marks the declaration-only store before its behavior exists.
var errNotImplemented = errors.New("interview progress store not implemented")

// Store persists interview progress through a shared GORM pool. Pins resolves the revision pin and
// lifecycle running instant that progress rows do not copy; Clock is the trusted application time.
type Store struct {
	DB    *gorm.DB
	Pins  interviewport.PinReader
	Clock service.Clock
}

var _ interviewport.ProgressStore = Store{}

// Insert is declared for the port and does nothing yet.
func (Store) Insert(context.Context, interviewport.ProgressWrite, interviewport.ProgressRecord) error {
	return errNotImplemented
}

// Find is declared for the port and does nothing yet.
func (Store) Find(context.Context, string) (interviewport.ProgressRecord, error) {
	return interviewport.ProgressRecord{}, errNotImplemented
}

// Update is declared for the port and does nothing yet.
func (Store) Update(context.Context, interviewport.ProgressWrite, func(*interviewport.ProgressRecord) error) error {
	return errNotImplemented
}
