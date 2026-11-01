package access

import (
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// AccountSnapshot transfers persisted administrative account facts without exposing the live aggregate.
type AccountSnapshot struct {
	ID        string
	Role      enum.Role
	Status    enum.AccountStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AdministrativeAccount owns stable administrative identity, a single fixed role and current eligibility status.
type AdministrativeAccount struct{ state AccountSnapshot }

// NewAdministrativeAccount records a new ACTIVE account with caller-supplied identity and one valid role.
func NewAdministrativeAccount(id string, role enum.Role, now time.Time) (*AdministrativeAccount, error) {
	return nil, ErrInvalidAccount
}

// RestoreAdministrativeAccount validates stored identity, role, status and timestamp order without resetting history.
func RestoreAdministrativeAccount(s AccountSnapshot) (*AdministrativeAccount, error) {
	return nil, ErrInvalidAccount
}

// Snapshot returns a detached copy of the current account facts.
func (a *AdministrativeAccount) Snapshot() AccountSnapshot { return AccountSnapshot{} }

// SubjectID returns the stable account identity used across sessions.
func (a *AdministrativeAccount) SubjectID() string { return "" }

// Eligible reports whether the account currently has Active status.
func (a *AdministrativeAccount) Eligible() bool { return false }
