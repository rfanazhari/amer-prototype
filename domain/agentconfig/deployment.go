package agentconfig

import (
	"errors"

	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
)

// ErrUnsupportedStoredTemplate reports a stored template identity outside the deployed restoration set.
var ErrUnsupportedStoredTemplate = errors.New("unsupported stored agent template version")

// SupportedRestoration returns catalog entries the running binary may load from persistence.
func SupportedRestoration() []Template {
	return Templates()
}

// RestorationSupported reports whether ref may be restored without current-creation reinterpretation.
func RestorationSupported(ref aggregate.TemplateRef) bool {
	for _, entry := range SupportedRestoration() {
		candidate := entry.Ref()
		if candidate.Template == ref.Template && candidate.Version == ref.Version && candidate.Behavior == ref.Behavior {
			return true
		}
	}
	return false
}

// SupportsCreation reports whether a new instance may be created for the exact template version.
func SupportsCreation(id, version string) bool {
	_, ok := Lookup(id, version)
	return ok
}
