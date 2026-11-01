package accessusecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// ProvisionHR creates an HR account under the current ADMIN authority.
type ProvisionHR struct {
	Accounts accessport.Accounts
	Current  service.CurrentAdministrativeSession
	Hasher   service.PasswordHasher
	Clock    service.Clock
}

// Execute persists a new HR identity after current ADMIN recheck and unique-email commit.
func (u ProvisionHR) Execute(ctx context.Context, name, email, password string) (*dto.AdministrativeAccount, error) {
	session, actor, err := u.Current.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	now := trustedNow(u.Clock)
	if err := authorizeCurrent(session, actor, now); err != nil {
		return nil, err
	}
	if !service.AllowsAccountProvisioning(actor.Role) {
		return nil, service.ErrForbidden
	}
	canonicalName, err := service.CanonicalAdministrativeName(name)
	if err != nil {
		return nil, err
	}
	canonicalEmail, err := service.CanonicalAdministrativeEmail(email)
	if err != nil {
		return nil, err
	}
	if err := service.AcceptProvisionPassword(password); err != nil {
		return nil, err
	}
	passwordHash, err := u.Hasher.Hash(ctx, password)
	if err != nil {
		return nil, err
	}
	account, err := access.NewAdministrativeAccount(uuid.NewString(), enum.HR, now)
	if err != nil {
		return nil, err
	}
	snapshot := account.Snapshot()
	createdBy := actor.ID
	record := accessport.AccountRecord{
		ID:           snapshot.ID,
		Name:         canonicalName,
		Email:        canonicalEmail,
		PasswordHash: passwordHash,
		Role:         snapshot.Role,
		Status:       snapshot.Status,
		CreatedBy:    &createdBy,
		CreatedAt:    snapshot.CreatedAt,
		UpdatedAt:    snapshot.UpdatedAt,
	}
	err = u.Accounts.WithinAccounts(ctx, func(tx accessport.AccountTx) error {
		current, found, err := tx.FindByEmail(ctx, actor.Email)
		if err != nil {
			return err
		}
		if !found || current.ID != actor.ID {
			return service.ErrSessionUnbound
		}
		if err := authorizeCurrent(session, current, now); err != nil {
			return err
		}
		if !service.AllowsAccountProvisioning(current.Role) {
			return service.ErrForbidden
		}
		return tx.Insert(ctx, record)
	})
	if err != nil {
		return nil, err
	}
	view := dto.FromAdministrativeAccount(record)
	return &view, nil
}
