package runtime

import (
	"net/http"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"gorm.io/gorm"
)

// NewCandidateInterview is not implemented yet.
func NewCandidateInterview(_ *gorm.DB, _ Config, _ *logmanager.Application) (http.Handler, http.Handler) {
	return nil, nil
}
