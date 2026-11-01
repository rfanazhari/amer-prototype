package agentconfighandler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentconfigusecase"
	catalog "github.com/rfanazhari/amer-prototype/domain/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

const jsonSafeMax = 9007199254740991

var (
	hrConfigKeys      = []string{"jobDescriptionRef", "mandatoryQuestions", "competencyTopics", "stageSeconds", "compensationPolicy"}
	publicationKeyPat = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

// createRequest accepts the approved create body without system-owned members.
type createRequest struct {
	TemplateID      string          `json:"templateId"`
	TemplateVersion string          `json:"templateVersion"`
	Name            string          `json:"name"`
	Configuration   json.RawMessage `json:"configuration"`
}

// replaceRequest accepts expected version and a complete configuration object.
type replaceRequest struct {
	ExpectedDraftVersion json.RawMessage `json:"expectedDraftVersion"`
	Configuration        json.RawMessage `json:"configuration"`
}

// validateRequest accepts the exact draft version to assess.
type validateRequest struct {
	ExpectedDraftVersion json.RawMessage `json:"expectedDraftVersion"`
}

// publishRequest accepts expected draft and validation identity; the key is a header.
type publishRequest struct {
	ExpectedDraftVersion json.RawMessage `json:"expectedDraftVersion"`
	ValidationID         string          `json:"validationId"`
}

// hrWire is the transport shape of an HR configuration object.
type hrWire struct {
	JobDescriptionRef  *valueobject.RefInput          `json:"jobDescriptionRef"`
	MandatoryQuestions []valueobject.QuestionInput    `json:"mandatoryQuestions"`
	CompetencyTopics   []string                       `json:"competencyTopics"`
	StageSeconds       *valueobject.StageSecondsInput `json:"stageSeconds"`
	CompensationPolicy *valueobject.CompensationInput `json:"compensationPolicy"`
}

// documentWire is the transport shape of a Document Analyzer configuration object.
type documentWire struct {
	RequiredFields []string `json:"requiredFields"`
}

// parseCreate maps a create body onto the authorized command input.
func parseCreate(r *http.Request) (agentconfigusecase.CreateInput, error) {
	var req createRequest
	if err := readJSON(r, &req, false); err != nil {
		return agentconfigusecase.CreateInput{}, err
	}
	if req.TemplateID == "" || req.TemplateVersion == "" || req.Name == "" || len(bytes.TrimSpace(req.Configuration)) == 0 {
		return agentconfigusecase.CreateInput{}, errInput
	}
	hr, fields, err := parseConfiguration(req.Configuration, false)
	if err != nil {
		return agentconfigusecase.CreateInput{}, err
	}
	in := agentconfigusecase.CreateInput{TemplateID: req.TemplateID, TemplateVersion: req.TemplateVersion, Name: req.Name}
	switch req.TemplateID {
	case catalog.TemplateDocumentAnalyzer:
		if hr != nil && (len(hr.MandatoryQuestions) > 0 || len(hr.CompetencyTopics) > 0 || hr.JobDescriptionRef != nil || hr.StageSeconds != nil || hr.CompensationPolicy != nil) {
			return agentconfigusecase.CreateInput{}, errInput
		}
		if fields == nil {
			fields = []string{}
		}
		in.DocumentFields = fields
	default:
		if fields != nil {
			return agentconfigusecase.CreateInput{}, errInput
		}
		if hr == nil {
			empty := valueobject.HRInput{}
			hr = &empty
		}
		in.HR = hr
	}
	return in, nil
}

// parseReplace maps a replacement body onto the authorized command input.
func parseReplace(r *http.Request, instanceID string) (agentconfigusecase.ReplaceInput, error) {
	var req replaceRequest
	if err := readJSON(r, &req, false); err != nil {
		return agentconfigusecase.ReplaceInput{}, err
	}
	version, err := parseDraftVersion(req.ExpectedDraftVersion)
	if err != nil {
		return agentconfigusecase.ReplaceInput{}, err
	}
	if len(bytes.TrimSpace(req.Configuration)) == 0 {
		return agentconfigusecase.ReplaceInput{}, errInput
	}
	hr, fields, err := parseConfiguration(req.Configuration, true)
	if err != nil {
		return agentconfigusecase.ReplaceInput{}, err
	}
	return agentconfigusecase.ReplaceInput{InstanceID: instanceID, ExpectedDraftVersion: version, HR: hr, DocumentFields: fields}, nil
}

// parseValidate maps a validation body onto the authorized command input.
func parseValidate(r *http.Request, instanceID string) (agentconfigusecase.ValidateInput, error) {
	var req validateRequest
	if err := readJSON(r, &req, false); err != nil {
		return agentconfigusecase.ValidateInput{}, err
	}
	version, err := parseDraftVersion(req.ExpectedDraftVersion)
	if err != nil {
		return agentconfigusecase.ValidateInput{}, err
	}
	return agentconfigusecase.ValidateInput{InstanceID: instanceID, ExpectedDraftVersion: version}, nil
}

// parsePublish maps a publication body and Idempotency-Key onto the authorized command input.
func parsePublish(r *http.Request, instanceID string) (agentconfigusecase.PublishInput, error) {
	var req publishRequest
	if err := readJSON(r, &req, false); err != nil {
		return agentconfigusecase.PublishInput{}, err
	}
	version, err := parseDraftVersion(req.ExpectedDraftVersion)
	if err != nil {
		return agentconfigusecase.PublishInput{}, err
	}
	if _, err = uuid.Parse(req.ValidationID); err != nil {
		return agentconfigusecase.PublishInput{}, errInput
	}
	key, err := publicationKey(r)
	if err != nil {
		return agentconfigusecase.PublishInput{}, err
	}
	return agentconfigusecase.PublishInput{InstanceID: instanceID, ValidationID: req.ValidationID, IdempotencyKey: key, ExpectedDraftVersion: version}, nil
}

// parseConfiguration classifies a configuration object as HR or Document without guessing extra fields.
func parseConfiguration(raw json.RawMessage, complete bool) (*valueobject.HRInput, []string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, nil, errInput
	}
	fields := map[string]json.RawMessage{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&fields); err != nil {
		return nil, nil, errInput
	}
	_, document := fields["requiredFields"]
	hrPresent := false
	for _, key := range hrConfigKeys {
		if _, ok := fields[key]; ok {
			hrPresent = true
			break
		}
	}
	if document && hrPresent {
		return nil, nil, errInput
	}
	for key := range fields {
		if key != "requiredFields" && !hrKey(key) {
			return nil, nil, errInput
		}
	}
	if document {
		var wire documentWire
		if err := decodeExact(raw, &wire); err != nil {
			return nil, nil, err
		}
		if wire.RequiredFields == nil {
			return nil, nil, errInput
		}
		for _, field := range wire.RequiredFields {
			if !enum.InvoiceField(field).Valid() {
				return nil, nil, errInput
			}
		}
		return nil, wire.RequiredFields, nil
	}
	if complete {
		for _, key := range hrConfigKeys {
			if _, ok := fields[key]; !ok {
				return nil, nil, errInput
			}
		}
	}
	var wire hrWire
	if err := decodeExact(raw, &wire); err != nil {
		return nil, nil, err
	}
	if wire.MandatoryQuestions == nil {
		wire.MandatoryQuestions = []valueobject.QuestionInput{}
	}
	if wire.CompetencyTopics == nil {
		wire.CompetencyTopics = []string{}
	}
	return &valueobject.HRInput{
		JobDescriptionRef: wire.JobDescriptionRef, MandatoryQuestions: wire.MandatoryQuestions,
		CompetencyTopics: wire.CompetencyTopics, StageSeconds: wire.StageSeconds, CompensationPolicy: wire.CompensationPolicy,
	}, nil, nil
}

// hrKey reports whether name is an approved HR configuration member.
func hrKey(name string) bool {
	for _, key := range hrConfigKeys {
		if key == name {
			return true
		}
	}
	return false
}

// decodeExact unmarshals one object and rejects unknown fields.
func decodeExact(raw json.RawMessage, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errInput
	}
	return nil
}

// parseDraftVersion accepts a JSON-safe positive integer draft version.
func parseDraftVersion(raw json.RawMessage) (uint64, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return 0, errInput
	}
	var number json.Number
	if json.Unmarshal(raw, &number) != nil {
		return 0, errInput
	}
	if _, err := strconv.ParseUint(number.String(), 10, 64); err != nil || bytes.ContainsRune([]byte(number.String()), '.') {
		if n, ferr := strconv.ParseFloat(number.String(), 64); ferr == nil && n > jsonSafeMax {
			return 0, service.ErrDraftConflict
		}
		return 0, errInput
	}
	value, err := strconv.ParseUint(number.String(), 10, 64)
	if err != nil || value == 0 {
		return 0, errInput
	}
	if value > jsonSafeMax {
		return 0, service.ErrDraftConflict
	}
	return value, nil
}

// publicationKey reads one untrimmed Idempotency-Key header value.
func publicationKey(r *http.Request) (string, error) {
	values := r.Header.Values("Idempotency-Key")
	if len(values) != 1 || !publicationKeyPat.MatchString(values[0]) {
		return "", errInput
	}
	return values[0], nil
}

// parseResourceID rejects malformed UUID path segments as invalid input.
func parseResourceID(value string) error {
	if _, err := uuid.Parse(value); err != nil {
		return errInput
	}
	return nil
}
