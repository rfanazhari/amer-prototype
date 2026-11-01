package agentconfigport

import "errors"

var (
	// ErrNotFound reports that an instance, revision or command was not stored.
	ErrNotFound = errors.New("agent configuration record not found")
	// ErrDuplicateCommand reports a unique command-identity loser that must reconcile outside the aborted transaction.
	ErrDuplicateCommand = errors.New("publication command identity already exists")
)
