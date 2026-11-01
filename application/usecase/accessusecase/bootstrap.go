package accessusecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// Bootstrap initializes the first ADMIN when applicable without overwriting existing accounts.
type Bootstrap struct {
	Accounts accessport.Accounts
	Hasher   service.PasswordHasher
	Clock    service.Clock
	Name     string
	Email    string
	Password string
}

// classifyBootstrapExisting skips compatible ACTIVE ADMIN state and fails incompatible rows.
func classifyBootstrapExisting(email string, existing accessport.AccountRecord, found bool) error {
	if !found {
		return nil
	}
	if existing.Email == email && existing.Role == enum.Admin && existing.Status == enum.Active {
		return nil
	}
	return service.ErrBootstrapIncompatible
}

// Execute hashes credentials outside the uniqueness transaction and inserts the first ADMIN when required.
func (u Bootstrap) Execute(ctx context.Context) error {
	name, err := service.CanonicalAdministrativeName(u.Name)
	if err != nil {
		return err
	}
	email, err := service.CanonicalAdministrativeEmail(u.Email)
	if err != nil {
		return err
	}
	if err := service.AcceptProvisionPassword(u.Password); err != nil {
		return err
	}
	now := trustedNow(u.Clock)
	passwordHash, err := u.Hasher.Hash(ctx, u.Password)
	if err != nil {
		return err
	}
	return u.Accounts.WithinAccounts(ctx, func(tx accessport.AccountTx) error {
		count, err := tx.Count(ctx)
		if err != nil {
			return err
		}
		if count > 0 {
			existing, found, err := tx.FindByEmail(ctx, email)
			if err != nil {
				return err
			}
			return classifyBootstrapExisting(email, existing, found)
		}
		account, err := access.NewAdministrativeAccount(uuid.NewString(), enum.Admin, now)
		if err != nil {
			return err
		}
		snapshot := account.Snapshot()
		err = tx.Insert(ctx, accessport.AccountRecord{
			ID:           snapshot.ID,
			Name:         name,
			Email:        email,
			PasswordHash: passwordHash,
			Role:         snapshot.Role,
			Status:       snapshot.Status,
			CreatedAt:    snapshot.CreatedAt,
			UpdatedAt:    snapshot.UpdatedAt,
		})
		if err == nil {
			return nil
		}
		var conflict accessport.AccountConflictError
		if !errors.As(err, &conflict) {
			return err
		}
		existing, found, findErr := tx.FindByEmail(ctx, email)
		if findErr != nil {
			return findErr
		}
		return classifyBootstrapExisting(email, existing, found)
	})
}
