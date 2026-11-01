package dto

import (
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
)

// FromAdministrativeAccount copies identity fields and omits password material.
func FromAdministrativeAccount(record accessport.AccountRecord) AdministrativeAccount {
	return AdministrativeAccount{
		ID:        record.ID,
		Name:      record.Name,
		Email:     record.Email,
		Role:      record.Role.String(),
		Status:    record.Status.String(),
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

// FromAdministrativeSession copies session metadata without the credential digest.
func FromAdministrativeSession(record accessport.SessionRecord) AdministrativeSession {
	return AdministrativeSession{ID: record.ID, ExpiresAt: record.ExpiresAt}
}

// AdministrativeAccount is a detached administrative identity without password material.
type AdministrativeAccount struct {
	ID        string
	Name      string
	Email     string
	Role      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AdministrativeSession is detached session metadata. The cookie remains the credential.
type AdministrativeSession struct {
	ID        string
	ExpiresAt time.Time
}

// AdministrativeLogin is the detached login result after a committed session insert.
type AdministrativeLogin struct {
	Account      AdministrativeAccount
	Session      AdministrativeSession
	SessionToken string
}

// AdministrativeCurrentSession is the detached current-session view including CSRF material.
type AdministrativeCurrentSession struct {
	Account   AdministrativeAccount
	Session   AdministrativeSession
	CSRFToken string
}
