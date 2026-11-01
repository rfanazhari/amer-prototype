package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
)

// RequireAgentConfigurationDeploymentCompatible rejects startup when stored template identities
// exceed the binary restoration set, preventing unsupported multi-version rollbacks.
func RequireAgentConfigurationDeploymentCompatible(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("deployment check requires database")
	}
	rows, err := db.QueryContext(ctx, storedTemplateIdentitySQL)
	if err != nil {
		return errors.New("agent configuration deployment unavailable")
	}
	defer func() { _ = rows.Close() }()
	unsupported := make([]string, 0)
	for rows.Next() {
		var id, version, behavior string
		if err = rows.Scan(&id, &version, &behavior); err != nil {
			return errors.New("unreadable agent configuration deployment metadata")
		}
		ref := aggregate.TemplateRef{Template: id, Version: version, Behavior: behavior}
		if !catalog.RestorationSupported(ref) {
			unsupported = append(unsupported, fmt.Sprintf("%s@%s behavior %s", id, version, behavior))
		}
	}
	if rows.Err() != nil {
		return errors.New("agent configuration deployment metadata read failed")
	}
	if len(unsupported) == 0 {
		return nil
	}
	return fmt.Errorf("incompatible agent configuration deployment; unsupported stored identities: %s", strings.Join(unsupported, ", "))
}

const storedTemplateIdentitySQL = `
SELECT template_id, template_version, behavior_version FROM public.agent_instances
UNION
SELECT template_id, template_version, behavior_version FROM public.agent_revisions
UNION
SELECT template_id, template_version, behavior_version FROM public.agent_validations
`
