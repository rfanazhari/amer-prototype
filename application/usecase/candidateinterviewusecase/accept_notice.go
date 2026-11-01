package candidateinterviewusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/candidateinterviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// AcceptNotice stores or finds the acceptance of the current notice version for the resolved candidate session.
type AcceptNotice struct {
	Current  service.CurrentCandidateSession
	Catalog  service.NoticeCatalog
	Tx       candidateinterviewport.ConsentTransaction
	Consents candidateinterviewport.Consents
	Clock    service.Clock
}

// Execute records the acceptance; this stub fails closed until the behavior is implemented.
func (u AcceptNotice) Execute(_ context.Context, _ AcceptNoticeInput) (*AcceptNoticeResult, error) {
	return nil, service.ErrInterviewDependencyUnavailable
}

var _ AcceptNoticeOperation = AcceptNotice{}
