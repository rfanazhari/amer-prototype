package dto

import "time"

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
	Account AdministrativeAccount
	Session AdministrativeSession
}

// AdministrativeCurrentSession is the detached current-session view including CSRF material.
type AdministrativeCurrentSession struct {
	Account   AdministrativeAccount
	Session   AdministrativeSession
	CSRFToken string
}
