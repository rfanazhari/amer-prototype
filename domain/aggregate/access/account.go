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
	if id == "" || !role.Valid() || now.IsZero() {
		return nil, ErrInvalidAccount
	}
	return &AdministrativeAccount{state: AccountSnapshot{
		ID:        id,
		Role:      role,
		Status:    enum.Active,
		CreatedAt: now,
		UpdatedAt: now,
	}}, nil
}

// RestoreAdministrativeAccount validates stored identity, role, status and timestamp order without resetting history.
func RestoreAdministrativeAccount(s AccountSnapshot) (*AdministrativeAccount, error) {
	if s.ID == "" || !s.Role.Valid() || !s.Status.Valid() || s.CreatedAt.IsZero() || s.UpdatedAt.Before(s.CreatedAt) {
		return nil, ErrInvalidAccount
	}
	return &AdministrativeAccount{state: s}, nil
}

// Snapshot returns a detached copy of the current account facts.
func (a *AdministrativeAccount) Snapshot() AccountSnapshot {
	if a == nil {
		return AccountSnapshot{}
	}
	return a.state
}

// SubjectID returns the stable account identity used across sessions.
func (a *AdministrativeAccount) SubjectID() string {
	if a == nil {
		return ""
	}
	return a.state.ID
}

// Eligible reports whether the account currently has Active status.
func (a *AdministrativeAccount) Eligible() bool {
	return a != nil && a.state.ID != "" && a.state.Role.Valid() && a.state.Status == enum.Active
}
