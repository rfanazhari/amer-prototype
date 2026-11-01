package agentconfig

import (
	"errors"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
)

var ErrUnknownTemplate = errors.New("unknown agent template")

const (
	TemplateHRInterviewer      = "hr-interviewer"
	TemplateDocumentAnalyzer   = "document-analyzer"
	TemplateVersion1           = "1"
	BehaviorVersion1           = "1"
	WarningPolicyInformational = "informational"
	DisplayHRInterviewer       = "HR Interviewer"
	DisplayDocumentAnalyzer    = "Document Analyzer"
)

// Template is a developer-owned catalog entry for one immutable template version.
type Template struct {
	ID, Version, Behavior, DisplayName, WarningPolicy string
}

// Templates returns both supported version-1 catalog entries.
func Templates() []Template { return nil }

// Lookup returns the catalog entry for an exact template identity and version.
func Lookup(id, version string) (Template, bool) { return Template{}, false }

// Ref returns the opaque template reference for a catalog entry.
func (t Template) Ref() agentconfig.TemplateRef {
	return agentconfig.TemplateRef{Template: t.ID, Version: t.Version, Behavior: t.Behavior}
}
