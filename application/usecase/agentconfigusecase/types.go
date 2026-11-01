package agentconfigusecase

import "github.com/rfanazhari/amer-prototype/domain/valueobject"

// CreateInput is the authorized create command after transport decoding.
type CreateInput struct {
	TemplateID, TemplateVersion, Name string
	HR                                *valueobject.HRInput
	DocumentFields                    []string
}

// ReplaceInput is the authorized full-replacement command after transport decoding.
type ReplaceInput struct {
	InstanceID           string
	ExpectedDraftVersion uint64
	HR                   *valueobject.HRInput
	DocumentFields       []string
}

// ValidateInput is the authorized exact-version validation command.
type ValidateInput struct {
	InstanceID           string
	ExpectedDraftVersion uint64
}

// PublishInput is the authorized publication command including the request key.
type PublishInput struct {
	InstanceID, ValidationID, IdempotencyKey string
	ExpectedDraftVersion                     uint64
}
