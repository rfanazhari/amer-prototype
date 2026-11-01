package agentconfig

import (
	"github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// AssessDocument records deterministic Document Analyzer findings without provider calls.
func AssessDocument(cfg valueobject.DocumentConfiguration) []agentconfig.Finding {
	return nil
}
