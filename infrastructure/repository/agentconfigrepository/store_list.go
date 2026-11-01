package agentconfigrepository

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"gorm.io/gorm"
)

// draftKey identifies the current draft row owned by one instance.
type draftKey struct {
	instanceID string
	version    int64
}

// ListInstances restores every instance in created_at descending, then id descending order.
func (s Store) ListInstances(ctx context.Context) ([]agentconfigport.StoredInstance, error) {
	var instances []InstanceModel
	err := s.DB.WithContext(ctx).Order("created_at DESC, id DESC").Find(&instances).Error
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return []agentconfigport.StoredInstance{}, nil
	}
	drafts, err := currentDrafts(ctx, s.DB, instances)
	if err != nil {
		return nil, err
	}
	revisions, err := activeRevisions(ctx, s.DB, instances)
	if err != nil {
		return nil, err
	}
	out := make([]agentconfigport.StoredInstance, 0, len(instances))
	for _, instance := range instances {
		draft, ok := drafts[draftKey{instanceID: instance.ID, version: instance.CurrentDraftVersion}]
		if !ok {
			return nil, technical(aggregate.ErrInvalid)
		}
		var active *RevisionModel
		if instance.ActiveRevisionID != nil {
			revision, found := revisions[*instance.ActiveRevisionID]
			if !found || revision.InstanceID != instance.ID {
				return nil, technical(aggregate.ErrInvalid)
			}
			active = &revision
		}
		stored, err := toStoredInstance(instance, draft, active)
		if err != nil {
			return nil, err
		}
		out = append(out, stored)
	}
	return out, nil
}

// currentDrafts loads only the draft row selected by each instance's current version.
func currentDrafts(ctx context.Context, db *gorm.DB, instances []InstanceModel) (map[draftKey]DraftModel, error) {
	ids := make([]string, len(instances))
	for i, instance := range instances {
		ids[i] = instance.ID
	}
	var rows []DraftModel
	err := db.WithContext(ctx).Model(&DraftModel{}).
		Select("agent_drafts.instance_id, agent_drafts.version, agent_drafts.configuration, agent_drafts.created_at, agent_drafts.updated_at").
		Joins("JOIN agent_instances ON agent_instances.id = agent_drafts.instance_id AND agent_instances.current_draft_version = agent_drafts.version").
		Where("agent_drafts.instance_id IN ?", ids).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[draftKey]DraftModel, len(rows))
	for _, row := range rows {
		out[draftKey{instanceID: row.InstanceID, version: row.Version}] = row
	}
	return out, nil
}

// activeRevisions loads the active-revision identities referenced by the listed instances.
func activeRevisions(ctx context.Context, db *gorm.DB, instances []InstanceModel) (map[string]RevisionModel, error) {
	revisionIDs := make([]string, 0)
	instanceIDs := make([]string, len(instances))
	for i, instance := range instances {
		instanceIDs[i] = instance.ID
		if instance.ActiveRevisionID != nil {
			revisionIDs = append(revisionIDs, *instance.ActiveRevisionID)
		}
	}
	out := map[string]RevisionModel{}
	if len(revisionIDs) == 0 {
		return out, nil
	}
	var rows []RevisionModel
	err := db.WithContext(ctx).Where("id IN ? AND instance_id IN ?", revisionIDs, instanceIDs).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}
