package agentconfigusecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Publish commits a new revision or returns a matching stored command result.
type Publish struct {
	Authority service.AgentConfigurationAuthority
	Instances agentconfigport.Store
	Reader    agentconfigport.InstanceReader
	Replay    agentconfigport.ReplayReader
	Clock     service.Clock
}

// Execute authorizes publication, checks replay before current-draft eligibility, and withholds results until commit.
func (u Publish) Execute(ctx context.Context, in PublishInput) (*dto.AgentRevision, error) {
	principal, err := authorize(ctx, u.Authority, service.AgentConfigurationPublish, in.InstanceID)
	if err != nil {
		return nil, err
	}
	if err := acceptPublicationKey(in.IdempotencyKey); err != nil {
		return nil, err
	}
	fingerprint, err := publicationFingerprint(in.InstanceID, in.ExpectedDraftVersion, in.ValidationID)
	if err != nil {
		return nil, err
	}
	identity := agentconfigport.CommandIdentity{Scope: principal.SubjectID, Kind: PublicationCommandKind, Key: in.IdempotencyKey}
	var result agentconfigport.CommandResult
	var replayed bool
	var created agentconfigport.StoredRevision
	err = u.Instances.WithinInstance(ctx, in.InstanceID, func(tx agentconfigport.InstanceTx) error {
		cmd, found, err := tx.FindCommand(ctx, identity)
		if err != nil {
			return err
		}
		if found {
			if cmd.Fingerprint != fingerprint {
				return service.ErrIdempotencyConflict
			}
			if err := u.Authority.Authorize(ctx, principal, service.AgentConfigurationReplay, in.InstanceID); err != nil {
				return err
			}
			replayed = true
			result = cmd
			return nil
		}
		return u.publishNew(ctx, tx, in, identity, fingerprint, &result, &created)
	})
	if err != nil {
		if errors.Is(err, agentconfigport.ErrDuplicateCommand) || isUncertain(err) {
			return u.reconcile(ctx, principal, identity, fingerprint, in.InstanceID, err)
		}
		return nil, err
	}
	if replayed {
		return u.loadRevision(ctx, in.InstanceID, result.RevisionID)
	}
	return dto.FromStoredRevision(created)
}

// publishNew verifies the locked draft and commits revision, active pointer and command together.
func (u Publish) publishNew(ctx context.Context, tx agentconfigport.InstanceTx, in PublishInput, identity agentconfigport.CommandIdentity, fingerprint string, result *agentconfigport.CommandResult, created *agentconfigport.StoredRevision) error {
	stored, err := tx.Instance(ctx)
	if err != nil {
		return err
	}
	draft, err := currentDraftNumber(stored)
	if err != nil {
		return err
	}
	if draft != in.ExpectedDraftVersion {
		return service.ErrDraftConflict
	}
	validation, err := tx.FindValidation(ctx, in.ValidationID)
	if err != nil {
		return err
	}
	revision, err := publishStored(&stored, uuid.NewString(), validation, trustedNow(u.Clock))
	if err != nil {
		return err
	}
	if err := tx.InsertRevision(ctx, revision); err != nil {
		return err
	}
	if err := tx.SaveInstance(ctx, stored); err != nil {
		return err
	}
	*result = agentconfigport.CommandResult{Fingerprint: fingerprint, InstanceID: in.InstanceID, RevisionID: revisionID(revision)}
	if err := tx.InsertCommand(ctx, identity, *result); err != nil {
		return err
	}
	*created = revision
	return nil
}

// reconcile reads the original command identity after unique conflict or uncertain commit.
func (u Publish) reconcile(ctx context.Context, principal service.AdminPrincipal, identity agentconfigport.CommandIdentity, fingerprint, instanceID string, original error) (*dto.AgentRevision, error) {
	cmd, found, err := u.Replay.FindCommand(ctx, identity)
	if err != nil || !found {
		return nil, original
	}
	if cmd.Fingerprint != fingerprint {
		return nil, service.ErrIdempotencyConflict
	}
	if err := u.Authority.Authorize(ctx, principal, service.AgentConfigurationReplay, instanceID); err != nil {
		return nil, err
	}
	return u.loadRevision(ctx, instanceID, cmd.RevisionID)
}

// loadRevision discloses an authorized detached revision after commit or successful replay.
func (u Publish) loadRevision(ctx context.Context, instanceID, revisionID string) (*dto.AgentRevision, error) {
	stored, err := u.Reader.FindRevision(ctx, instanceID, revisionID)
	if err != nil {
		return nil, err
	}
	return dto.FromStoredRevision(stored)
}

// revisionID returns the stored revision identity from the template-tagged unit.
func revisionID(stored agentconfigport.StoredRevision) string {
	if stored.HR != nil {
		return stored.HR.Snapshot().Identity.ID
	}
	if stored.Document != nil {
		return stored.Document.Snapshot().Identity.ID
	}
	return ""
}

// isUncertain reports a commit whose acceptance must be reconciled by the original key.
func isUncertain(err error) bool {
	var uncertain *agentconfigport.CommitUncertainError
	return errors.As(err, &uncertain)
}
