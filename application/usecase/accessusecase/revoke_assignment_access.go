package accessusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
)

// RevokeAssignmentAccess withdraws assignment access under current HR authority and cancels a nonterminal run.
type RevokeAssignmentAccess struct {
	Assignments accessport.AssignmentAccess
	Current     service.CurrentAdministrativeSession
	Lifecycle   executionusecase.CancelNonterminalOperation
	Clock       service.Clock
}

var _ RevokeAssignmentAccessOperation = RevokeAssignmentAccess{}

// Execute is not implemented yet.
func (u RevokeAssignmentAccess) Execute(context.Context, string) (*dto.AssignmentRevocation, error) {
	return nil, nil
}
