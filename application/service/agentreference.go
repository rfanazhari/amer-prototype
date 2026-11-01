package service

import "errors"

// ErrReferenceIdentity reports a reader result whose identity is not the requested version.
var ErrReferenceIdentity = errors.New("resolved reference identity does not match the request")

const (
	// ReferenceKindJobDescription identifies an immutable synthetic job-description version.
	ReferenceKindJobDescription = "job-description"
	// ReferenceKindCompensationPolicy identifies an immutable confidential compensation-policy version.
	ReferenceKindCompensationPolicy = "compensation-policy"
	// DemoJobRole is the locally required job-description role for HR publication.
	DemoJobRole = "Backend Engineer"
)

// AgentReferenceFact is a detached immutable reference version resolved outside the domain.
type AgentReferenceFact struct {
	Kind, ID, Version, Role string
	Ready                   bool
}
