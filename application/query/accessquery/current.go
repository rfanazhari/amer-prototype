package accessquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// CurrentSessionOperation returns the current administrative session after trusted binding.
type CurrentSessionOperation interface {
	// Execute returns detached account, session and CSRF material for a current bound session.
	Execute(ctx context.Context) (*dto.AdministrativeCurrentSession, error)
}

// CurrentSession returns the current administrative session after trusted binding.
type CurrentSession struct {
	Current service.CurrentAdministrativeSession
	CSRF    service.CSRFTokens
	Clock   service.Clock
}

// Execute is an inert T6 RED stub; GREEN returns CSRF material only for a current bound session.
func (CurrentSession) Execute(context.Context) (*dto.AdministrativeCurrentSession, error) {
	return nil, nil
}
