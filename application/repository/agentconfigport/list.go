package agentconfigport

import "context"

// InstanceLister loads every detached instance for one authorized collection read.
type InstanceLister interface {
	// ListInstances returns every restored instance in created_at descending, then id descending order.
	ListInstances(context.Context) ([]StoredInstance, error)
}
