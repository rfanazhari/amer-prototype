package access

import "errors"

var (
	// ErrInvalidAccount reports rejected administrative account construction or restoration.
	ErrInvalidAccount = errors.New("invalid administrative account")
	// ErrInvalidSession reports rejected administrative session construction, restoration or revocation.
	ErrInvalidSession = errors.New("invalid administrative session")
	// ErrInvalidCandidate reports rejected candidate identity construction or restoration.
	ErrInvalidCandidate = errors.New("invalid candidate")
	// ErrInvalidInvitation reports rejected invitation construction, restoration or revocation.
	ErrInvalidInvitation = errors.New("invalid invitation")
	// ErrInvalidCandidateSession reports rejected candidate session construction or restoration.
	ErrInvalidCandidateSession = errors.New("invalid candidate session")
)
