package accessauthority

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Sessions resolves the current administrative session from a trusted context credential.
type Sessions struct {
	Store  accessport.Sessions
	Tokens service.TokenSecrets
	Clock  service.Clock
}

// Resolve is an inert T7 RED stub; GREEN binds digest lookup to current account eligibility.
func (Sessions) Resolve(context.Context) (accessport.SessionRecord, accessport.AccountRecord, error) {
	return accessport.SessionRecord{}, accessport.AccountRecord{}, nil
}
