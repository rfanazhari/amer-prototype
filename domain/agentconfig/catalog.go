package agentconfig

import (
	"errors"

	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
)

// ErrUnknownTemplate reports a template identity or behavior version absent from the catalog.
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
func Templates() []Template {
	return []Template{
		{ID: TemplateHRInterviewer, Version: TemplateVersion1, Behavior: BehaviorVersion1, DisplayName: DisplayHRInterviewer, WarningPolicy: WarningPolicyInformational},
		{ID: TemplateDocumentAnalyzer, Version: TemplateVersion1, Behavior: BehaviorVersion1, DisplayName: DisplayDocumentAnalyzer, WarningPolicy: WarningPolicyInformational},
	}
}

// Lookup returns the catalog entry for an exact template identity and version.
func Lookup(id, version string) (Template, bool) {
	for _, template := range Templates() {
		if template.ID == id && template.Version == version {
			return template, true
		}
	}
	return Template{}, false
}

// Ref returns the opaque template reference for a catalog entry.
func (t Template) Ref() aggregate.TemplateRef {
	return aggregate.TemplateRef{Template: t.ID, Version: t.Version, Behavior: t.Behavior}
}
