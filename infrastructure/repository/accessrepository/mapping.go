package accessrepository

import (
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// copyString returns an independent optional identity so callers cannot mutate stored creator aliases.
func copyString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

// copyTime returns an independent optional instant so callers cannot mutate stored revocation aliases.
func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	canonical := value.UTC()
	return &canonical
}

// fromAccount maps a detached account record onto the storage model.
func fromAccount(record accessport.AccountRecord) AccountModel {
	return AccountModel{
		ID:           record.ID,
		Name:         record.Name,
		Email:        record.Email,
		PasswordHash: record.PasswordHash,
		Role:         record.Role.String(),
		Status:       record.Status.String(),
		CreatedBy:    copyString(record.CreatedBy),
		CreatedAt:    record.CreatedAt.UTC(),
		UpdatedAt:    record.UpdatedAt.UTC(),
	}
}

// toAccount restores a detached account record and classifies corrupt stored facts as technical failures.
func toAccount(model AccountModel) (accessport.AccountRecord, error) {
	role := enum.Role(model.Role)
	status := enum.AccountStatus(model.Status)
	if _, err := access.RestoreAdministrativeAccount(access.AccountSnapshot{
		ID:        model.ID,
		Role:      role,
		Status:    status,
		CreatedAt: model.CreatedAt.UTC(),
		UpdatedAt: model.UpdatedAt.UTC(),
	}); err != nil {
		return accessport.AccountRecord{}, errs.Technical{Cause: err}
	}
	if model.Name == "" || model.Email == "" || model.PasswordHash == "" {
		return accessport.AccountRecord{}, errs.Technical{Cause: access.ErrInvalidAccount}
	}
	return accessport.AccountRecord{
		ID:           model.ID,
		Name:         model.Name,
		Email:        model.Email,
		PasswordHash: model.PasswordHash,
		Role:         role,
		Status:       status,
		CreatedBy:    copyString(model.CreatedBy),
		CreatedAt:    model.CreatedAt.UTC(),
		UpdatedAt:    model.UpdatedAt.UTC(),
	}, nil
}

// fromSession maps a detached session record onto the storage model.
func fromSession(record accessport.SessionRecord) SessionModel {
	return SessionModel{
		ID:        record.ID,
		AccountID: record.AccountID,
		TokenHash: record.TokenHash,
		CreatedAt: record.CreatedAt.UTC(),
		ExpiresAt: record.ExpiresAt.UTC(),
		RevokedAt: copyTime(record.RevokedAt),
	}
}

// toSession restores a detached session record and classifies corrupt stored facts as technical failures.
func toSession(model SessionModel) (accessport.SessionRecord, error) {
	if _, err := access.RestoreAdministrativeSession(access.SessionSnapshot{
		ID:        model.ID,
		AccountID: model.AccountID,
		CreatedAt: model.CreatedAt.UTC(),
		ExpiresAt: model.ExpiresAt.UTC(),
		RevokedAt: copyTime(model.RevokedAt),
	}); err != nil {
		return accessport.SessionRecord{}, errs.Technical{Cause: err}
	}
	if model.TokenHash == "" {
		return accessport.SessionRecord{}, errs.Technical{Cause: access.ErrInvalidSession}
	}
	return accessport.SessionRecord{
		ID:        model.ID,
		AccountID: model.AccountID,
		TokenHash: model.TokenHash,
		CreatedAt: model.CreatedAt.UTC(),
		ExpiresAt: model.ExpiresAt.UTC(),
		RevokedAt: copyTime(model.RevokedAt),
	}, nil
}

// fromCandidate maps a detached candidate record onto the storage model.
func fromCandidate(record accessport.CandidateRecord) CandidateModel {
	return CandidateModel{
		ID:        record.ID,
		Name:      record.Name,
		Email:     record.Email,
		CreatedBy: record.CreatedBy,
		CreatedAt: record.CreatedAt.UTC(),
		UpdatedAt: record.UpdatedAt.UTC(),
	}
}

// toCandidate restores a detached candidate record and classifies corrupt stored facts as technical failures.
func toCandidate(model CandidateModel) (accessport.CandidateRecord, error) {
	if _, err := access.RestoreCandidate(access.CandidateSnapshot{
		ID:        model.ID,
		Name:      model.Name,
		Email:     model.Email,
		CreatedBy: model.CreatedBy,
		CreatedAt: model.CreatedAt.UTC(),
		UpdatedAt: model.UpdatedAt.UTC(),
	}); err != nil {
		return accessport.CandidateRecord{}, errs.Technical{Cause: err}
	}
	return accessport.CandidateRecord{
		ID:        model.ID,
		Name:      model.Name,
		Email:     model.Email,
		CreatedBy: model.CreatedBy,
		CreatedAt: model.CreatedAt.UTC(),
		UpdatedAt: model.UpdatedAt.UTC(),
	}, nil
}

// fromInvitation maps a detached invitation record onto the storage model.
func fromInvitation(record accessport.InvitationRecord) InvitationModel {
	return InvitationModel{
		ID:           record.ID,
		AssignmentID: record.AssignmentID,
		TokenHash:    record.TokenHash,
		CreatedBy:    record.CreatedBy,
		CreatedAt:    record.CreatedAt.UTC(),
		ExpiresAt:    record.ExpiresAt.UTC(),
		RevokedAt:    copyTime(record.RevokedAt),
	}
}

// toInvitation restores a detached invitation record and classifies corrupt stored facts as technical failures.
func toInvitation(model InvitationModel) (accessport.InvitationRecord, error) {
	if _, err := access.RestoreInvitation(access.InvitationSnapshot{
		ID:           model.ID,
		AssignmentID: model.AssignmentID,
		CreatedBy:    model.CreatedBy,
		CreatedAt:    model.CreatedAt.UTC(),
		ExpiresAt:    model.ExpiresAt.UTC(),
		RevokedAt:    copyTime(model.RevokedAt),
	}); err != nil {
		return accessport.InvitationRecord{}, errs.Technical{Cause: err}
	}
	if model.TokenHash == "" {
		return accessport.InvitationRecord{}, errs.Technical{Cause: access.ErrInvalidInvitation}
	}
	return accessport.InvitationRecord{
		ID:           model.ID,
		AssignmentID: model.AssignmentID,
		TokenHash:    model.TokenHash,
		CreatedBy:    model.CreatedBy,
		CreatedAt:    model.CreatedAt.UTC(),
		ExpiresAt:    model.ExpiresAt.UTC(),
		RevokedAt:    copyTime(model.RevokedAt),
	}, nil
}

// toCandidateSession restores a detached candidate session record and classifies corrupt stored facts as technical failures.
func toCandidateSession(model CandidateSessionModel) (accessport.CandidateSessionRecord, error) {
	if _, err := access.RestoreCandidateSession(access.CandidateSessionSnapshot{
		ID:           model.ID,
		AssignmentID: model.AssignmentID,
		InvitationID: model.InvitationID,
		CreatedAt:    model.CreatedAt.UTC(),
		ExpiresAt:    model.ExpiresAt.UTC(),
		RevokedAt:    copyTime(model.RevokedAt),
	}); err != nil {
		return accessport.CandidateSessionRecord{}, errs.Technical{Cause: err}
	}
	if model.TokenHash == "" {
		return accessport.CandidateSessionRecord{}, errs.Technical{Cause: access.ErrInvalidCandidateSession}
	}
	return accessport.CandidateSessionRecord{
		ID:           model.ID,
		AssignmentID: model.AssignmentID,
		InvitationID: model.InvitationID,
		TokenHash:    model.TokenHash,
		CreatedAt:    model.CreatedAt.UTC(),
		ExpiresAt:    model.ExpiresAt.UTC(),
		RevokedAt:    copyTime(model.RevokedAt),
	}, nil
}

// fromCandidateSession maps a detached candidate session record onto the storage model.
func fromCandidateSession(record accessport.CandidateSessionRecord) CandidateSessionModel {
	return CandidateSessionModel{
		ID:           record.ID,
		AssignmentID: record.AssignmentID,
		InvitationID: record.InvitationID,
		TokenHash:    record.TokenHash,
		CreatedAt:    record.CreatedAt.UTC(),
		ExpiresAt:    record.ExpiresAt.UTC(),
		RevokedAt:    copyTime(record.RevokedAt),
	}
}
