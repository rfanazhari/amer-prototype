package agentconfigport

import (
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// StoredInstance is the template-tagged persistence unit for one instance and its display name.
type StoredInstance struct {
	Name     string
	HR       *aggregate.AgentInstance[valueobject.HRConfiguration]
	Document *aggregate.AgentInstance[valueobject.DocumentConfiguration]
}

// StoredRevision is the template-tagged persistence unit for one immutable revision.
type StoredRevision struct {
	HR       *aggregate.PublishedRevision[valueobject.HRConfiguration]
	Document *aggregate.PublishedRevision[valueobject.DocumentConfiguration]
}

// Store is the T6 instantiation of the configuration mutation port.
type Store = Mutation[StoredInstance, aggregate.Validation, StoredRevision]

// InstanceReader is the T6 instantiation of the bounded configuration read port.
type InstanceReader = Reader[StoredInstance, StoredRevision]

// InstanceTx is the T6 instantiation of the instance-owned transaction port.
type InstanceTx = Transaction[StoredInstance, aggregate.Validation, StoredRevision]
