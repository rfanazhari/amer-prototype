package agentconfigrepository

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// JSONDocument stores JSONB payloads without relying on ORM omission of empty documents.
type JSONDocument []byte

// Value returns the JSON bytes for the driver.
func (j JSONDocument) Value() (driver.Value, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

// Scan copies a driver JSONB value into the document.
func (j *JSONDocument) Scan(value any) error {
	switch typed := value.(type) {
	case nil:
		*j = nil
		return nil
	case []byte:
		*j = append([]byte(nil), typed...)
		return nil
	case string:
		*j = []byte(typed)
		return nil
	default:
		return fmt.Errorf("unsupported json document %T", value)
	}
}

// GormDataType reports the GORM generic JSON type.
func (JSONDocument) GormDataType() string { return "jsonb" }

// InstanceModel maps agent_instances independently of domain and API types.
type InstanceModel struct {
	ID                  string `gorm:"type:uuid;primaryKey"`
	Name                string `gorm:"type:text;not null"`
	TemplateID          string `gorm:"column:template_id;type:text;not null"`
	TemplateVersion     string `gorm:"column:template_version;type:text;not null"`
	BehaviorVersion     string `gorm:"column:behavior_version;type:text;not null"`
	CurrentDraftVersion int64  `gorm:"column:current_draft_version;not null"`
	ActiveRevisionID    *string
	CreatedAt           time.Time `gorm:"not null;autoCreateTime:false"`
	UpdatedAt           time.Time `gorm:"not null;autoUpdateTime:false"`
}

// TableName fixes the instance table used by GORM.
func (InstanceModel) TableName() string { return "agent_instances" }

// DraftModel maps agent_drafts independently of domain and API types.
type DraftModel struct {
	InstanceID    string `gorm:"type:uuid;primaryKey"`
	Version       int64  `gorm:"primaryKey;not null"`
	Configuration JSONDocument
	CreatedAt     time.Time `gorm:"not null;autoCreateTime:false"`
	UpdatedAt     time.Time `gorm:"not null;autoUpdateTime:false"`
}

// TableName fixes the draft table used by GORM.
func (DraftModel) TableName() string { return "agent_drafts" }

// ValidationModel maps agent_validations independently of domain and API types.
type ValidationModel struct {
	ID              string `gorm:"type:uuid;primaryKey"`
	InstanceID      string `gorm:"column:instance_id;type:uuid;not null"`
	DraftVersion    int64  `gorm:"column:draft_version;not null"`
	TemplateID      string `gorm:"column:template_id;type:text;not null"`
	TemplateVersion string `gorm:"column:template_version;type:text;not null"`
	BehaviorVersion string `gorm:"column:behavior_version;type:text;not null"`
	Eligible        bool   `gorm:"not null"`
	Findings        JSONDocument
	CreatedAt       time.Time `gorm:"not null;autoCreateTime:false"`
}

// TableName fixes the validation table used by GORM.
func (ValidationModel) TableName() string { return "agent_validations" }

// RevisionModel maps agent_revisions independently of domain and API types.
type RevisionModel struct {
	ID                 string `gorm:"type:uuid;primaryKey"`
	InstanceID         string `gorm:"column:instance_id;type:uuid;not null"`
	RevisionNumber     int64  `gorm:"column:revision_number;not null"`
	SourceDraftVersion int64  `gorm:"column:source_draft_version;not null"`
	ValidationID       string `gorm:"column:validation_id;type:uuid;not null"`
	TemplateID         string `gorm:"column:template_id;type:text;not null"`
	TemplateVersion    string `gorm:"column:template_version;type:text;not null"`
	BehaviorVersion    string `gorm:"column:behavior_version;type:text;not null"`
	Configuration      JSONDocument
	CreatedAt          time.Time `gorm:"not null;autoCreateTime:false"`
}

// TableName fixes the revision table used by GORM.
func (RevisionModel) TableName() string { return "agent_revisions" }

// CommandModel maps agent_publication_commands independently of domain and API types.
type CommandModel struct {
	Scope               string    `gorm:"type:uuid;primaryKey"`
	CommandKind         string    `gorm:"column:command_kind;primaryKey"`
	RequestKey          string    `gorm:"column:request_key;primaryKey"`
	IntentFingerprint   string    `gorm:"column:intent_fingerprint;type:text;not null"`
	ResultingRevisionID string    `gorm:"column:resulting_revision_id;type:uuid;not null"`
	CreatedAt           time.Time `gorm:"not null;autoCreateTime:false"`
}

// TableName fixes the publication-command table used by GORM.
func (CommandModel) TableName() string { return "agent_publication_commands" }
