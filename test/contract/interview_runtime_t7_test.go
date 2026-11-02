package contract_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenAPIOmitsPrototypeVoiceRoutes verifies the removed prototype voice routes and tag are not published.
func TestOpenAPIOmitsPrototypeVoiceRoutes(t *testing.T) {
	doc := loadContract(t)
	for _, path := range []string{"/voice-sessions", "/voice-session"} {
		t.Run(path, func(t *testing.T) {
			require.Nil(t, doc.Paths.Find(path))
		})
	}
	for _, tag := range doc.Tags {
		require.NotEqual(t, "voice-sessions", tag.Name)
	}
	for path, item := range doc.Paths.Map() {
		for _, op := range item.Operations() {
			require.NotContains(t, op.Tags, "voice-sessions", path)
		}
	}
}
