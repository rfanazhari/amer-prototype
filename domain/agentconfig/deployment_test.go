package agentconfig_test

import (
	"testing"

	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/stretchr/testify/assert"
)

// TestRestorationSupportedAcceptsRetainedVersionOneBehavior proves the current deployment retains v1 restoration.
func TestRestorationSupportedAcceptsRetainedVersionOneBehavior(t *testing.T) {
	for _, entry := range catalog.SupportedRestoration() {
		assert.True(t, catalog.RestorationSupported(entry.Ref()), entry.ID)
	}
}

// TestRestorationSupportedRejectsUnknownBehaviorVersion prevents silent loading of future behavior tokens.
func TestRestorationSupportedRejectsUnknownBehaviorVersion(t *testing.T) {
	ref := aggregate.TemplateRef{
		Template: catalog.TemplateHRInterviewer,
		Version:  catalog.TemplateVersion1,
		Behavior: "2",
	}
	assert.False(t, catalog.RestorationSupported(ref))
}
