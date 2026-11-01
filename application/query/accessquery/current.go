package accessquery

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
)

// CurrentSessionOperation returns the current administrative session after trusted binding.
type CurrentSessionOperation interface {
	// Execute returns detached account, session and CSRF material for a current bound session.
	Execute(ctx context.Context) (*dto.AdministrativeCurrentSession, error)
}
