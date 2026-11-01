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
