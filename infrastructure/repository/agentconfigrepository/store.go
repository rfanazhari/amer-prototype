package agentconfigrepository

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"gorm.io/gorm"
)

var errUnimplemented = errors.New("agent configuration persistence is not implemented")

// Store implements configuration persistence through a runtime-owned GORM pool.
type Store struct{ DB *gorm.DB }

var (
	_ agentconfigport.Store          = Store{}
	_ agentconfigport.InstanceReader = Store{}
	_ agentconfigport.ReplayReader   = Store{}
	_ agentconfigport.InstanceTx     = instanceTx{}
)

// instanceTx is the instance-owned transaction surface used after the instance row lock.
type instanceTx struct{}

// Create is inert until the PostgreSQL adapter is implemented.
func (s Store) Create(context.Context, agentconfigport.StoredInstance) error {
	return errUnimplemented
}

// FindInstance is inert until the PostgreSQL adapter is implemented.
func (s Store) FindInstance(context.Context, string) (agentconfigport.StoredInstance, error) {
	return agentconfigport.StoredInstance{}, errUnimplemented
}

// FindRevision is inert until the PostgreSQL adapter is implemented.
func (s Store) FindRevision(context.Context, string, string) (agentconfigport.StoredRevision, error) {
	return agentconfigport.StoredRevision{}, errUnimplemented
}

// FindCommand is inert until the PostgreSQL adapter is implemented.
func (s Store) FindCommand(context.Context, agentconfigport.CommandIdentity) (agentconfigport.CommandResult, bool, error) {
	return agentconfigport.CommandResult{}, false, errUnimplemented
}

// WithinInstance is inert until the PostgreSQL adapter is implemented.
func (s Store) WithinInstance(context.Context, string, func(agentconfigport.InstanceTx) error) error {
	return errUnimplemented
}

// Instance is inert until the PostgreSQL adapter is implemented.
func (instanceTx) Instance(context.Context) (agentconfigport.StoredInstance, error) {
	return agentconfigport.StoredInstance{}, errUnimplemented
}

// SaveInstance is inert until the PostgreSQL adapter is implemented.
func (instanceTx) SaveInstance(context.Context, agentconfigport.StoredInstance) error {
	return errUnimplemented
}

// FindValidation is inert until the PostgreSQL adapter is implemented.
func (instanceTx) FindValidation(context.Context, string) (aggregate.Validation, error) {
	return aggregate.Validation{}, errUnimplemented
}

// InsertValidation is inert until the PostgreSQL adapter is implemented.
func (instanceTx) InsertValidation(context.Context, aggregate.Validation) error {
	return errUnimplemented
}

// FindCommand is inert until the PostgreSQL adapter is implemented.
func (instanceTx) FindCommand(context.Context, agentconfigport.CommandIdentity) (agentconfigport.CommandResult, bool, error) {
	return agentconfigport.CommandResult{}, false, errUnimplemented
}

// InsertRevision is inert until the PostgreSQL adapter is implemented.
func (instanceTx) InsertRevision(context.Context, agentconfigport.StoredRevision) error {
	return errUnimplemented
}

// InsertCommand is inert until the PostgreSQL adapter is implemented.
func (instanceTx) InsertCommand(context.Context, agentconfigport.CommandIdentity, agentconfigport.CommandResult) error {
	return errUnimplemented
}
