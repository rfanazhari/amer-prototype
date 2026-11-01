package agentconfigusecase

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/rfanazhari/amer-prototype/application/service"
)

const (
	// PublicationCommandKind is the durable publication command kind approved in design §6.4.
	PublicationCommandKind = "publish-agent-configuration/v1"
)

var publicationKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// acceptPublicationKey accepts the approved case-sensitive request key without trimming.
func acceptPublicationKey(key string) error {
	if !publicationKeyPattern.MatchString(key) {
		return service.ErrInvalidInput
	}
	return nil
}

// publicationFingerprint hashes the approved compact JSON intent array.
func publicationFingerprint(instanceID string, draftVersion uint64, validationID string) (string, error) {
	payload, err := json.Marshal([]any{PublicationCommandKind, strings.ToLower(instanceID), draftVersion, strings.ToLower(validationID)})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
