package agentconfigrepository

import (
	"encoding/json"

	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// refPayload is the stored identity and version of a stable reference.
type refPayload struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// questionPayload is one stored mandatory question.
type questionPayload struct {
	ID    string `json:"id"`
	Stage string `json:"stage"`
	Text  string `json:"text"`
}

// stagePayload is the stored four-stage duration object.
type stagePayload struct {
	Background   int `json:"background"`
	Screening    int `json:"screening"`
	Compensation int `json:"compensation"`
	Closing      int `json:"closing"`
}

// compensationPayload is the stored compensation discussion policy.
type compensationPayload struct {
	DiscussionMode   string      `json:"discussionMode"`
	SalaryCeilingRef *refPayload `json:"salaryCeilingRef"`
}

// hrPayload is the JSONB body for an HR draft or revision.
type hrPayload struct {
	JobDescriptionRef  *refPayload          `json:"jobDescriptionRef"`
	MandatoryQuestions []questionPayload    `json:"mandatoryQuestions"`
	CompetencyTopics   []string             `json:"competencyTopics"`
	StageSeconds       *stagePayload        `json:"stageSeconds"`
	CompensationPolicy *compensationPayload `json:"compensationPolicy"`
}

// documentPayload is the JSONB body for a Document Analyzer draft or revision.
type documentPayload struct {
	RequiredFields []string `json:"requiredFields"`
}

// findingPayload is one stored validation finding.
type findingPayload struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

// encodeHRConfiguration writes explicit nulls and empty arrays for a stored HR draft.
func encodeHRConfiguration(cfg valueobject.HRConfiguration) (JSONDocument, error) {
	payload := hrPayload{MandatoryQuestions: []questionPayload{}, CompetencyTopics: []string{}}
	if ref := cfg.JobDescriptionRef(); ref != nil {
		payload.JobDescriptionRef = &refPayload{ID: ref.ID(), Version: ref.Version()}
	}
	for _, question := range cfg.MandatoryQuestions() {
		payload.MandatoryQuestions = append(payload.MandatoryQuestions, questionPayload{ID: question.ID(), Stage: question.Stage().String(), Text: question.Text()})
	}
	payload.CompetencyTopics = append([]string{}, cfg.CompetencyTopics()...)
	if stages := cfg.StageSeconds(); stages != nil {
		payload.StageSeconds = &stagePayload{Background: stages.Background(), Screening: stages.Screening(), Compensation: stages.Compensation(), Closing: stages.Closing()}
	}
	if policy := cfg.CompensationPolicy(); policy != nil {
		item := &compensationPayload{DiscussionMode: policy.DiscussionMode()}
		if ceiling := policy.SalaryCeilingRef(); ceiling != nil {
			item.SalaryCeilingRef = &refPayload{ID: ceiling.ID(), Version: ceiling.Version()}
		}
		payload.CompensationPolicy = item
	}
	body, err := json.Marshal(payload)
	return JSONDocument(body), err
}

// decodeHRConfiguration restores an HR draft from stored JSON without live provider checks.
func decodeHRConfiguration(body JSONDocument) (valueobject.HRConfiguration, error) {
	var payload hrPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return valueobject.HRConfiguration{}, err
	}
	input := valueobject.HRInput{CompetencyTopics: payload.CompetencyTopics}
	if payload.JobDescriptionRef != nil {
		input.JobDescriptionRef = &valueobject.RefInput{ID: payload.JobDescriptionRef.ID, Version: payload.JobDescriptionRef.Version}
	}
	for _, question := range payload.MandatoryQuestions {
		input.MandatoryQuestions = append(input.MandatoryQuestions, valueobject.QuestionInput{ID: question.ID, Stage: question.Stage, Text: question.Text})
	}
	if payload.StageSeconds != nil {
		input.StageSeconds = &valueobject.StageSecondsInput{Background: payload.StageSeconds.Background, Screening: payload.StageSeconds.Screening, Compensation: payload.StageSeconds.Compensation, Closing: payload.StageSeconds.Closing}
	}
	if payload.CompensationPolicy != nil {
		policy := &valueobject.CompensationInput{DiscussionMode: payload.CompensationPolicy.DiscussionMode}
		if payload.CompensationPolicy.SalaryCeilingRef != nil {
			policy.SalaryCeilingRef = &valueobject.RefInput{ID: payload.CompensationPolicy.SalaryCeilingRef.ID, Version: payload.CompensationPolicy.SalaryCeilingRef.Version}
		}
		input.CompensationPolicy = policy
	}
	return valueobject.NewHRConfiguration(input)
}

// encodeDocumentConfiguration writes a catalog-ordered required-field list.
func encodeDocumentConfiguration(cfg valueobject.DocumentConfiguration) (JSONDocument, error) {
	payload := documentPayload{RequiredFields: []string{}}
	for _, field := range cfg.RequiredFields() {
		payload.RequiredFields = append(payload.RequiredFields, field.String())
	}
	body, err := json.Marshal(payload)
	return JSONDocument(body), err
}

// decodeDocumentConfiguration restores a Document draft from stored JSON.
func decodeDocumentConfiguration(body JSONDocument) (valueobject.DocumentConfiguration, error) {
	var payload documentPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return valueobject.DocumentConfiguration{}, err
	}
	return valueobject.NewDocumentConfiguration(payload.RequiredFields)
}

// encodeFindings writes an explicit empty array when no findings were recorded.
func encodeFindings(findings []aggregate.Finding) (JSONDocument, error) {
	payload := make([]findingPayload, 0, len(findings))
	for _, finding := range findings {
		payload = append(payload, findingPayload{Severity: finding.Severity.String(), Code: finding.Code, Path: finding.Path, Message: finding.Message})
	}
	body, err := json.Marshal(payload)
	return JSONDocument(body), err
}

// decodeFindings restores assessment findings and rejects unknown severities.
func decodeFindings(body JSONDocument) ([]aggregate.Finding, error) {
	if len(body) == 0 {
		return []aggregate.Finding{}, nil
	}
	var payload []findingPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	out := make([]aggregate.Finding, 0, len(payload))
	for _, item := range payload {
		severity := enum.FindingSeverity(item.Severity)
		if item.Severity != "" && !severity.Valid() {
			return nil, aggregate.ErrInvalid
		}
		out = append(out, aggregate.Finding{Severity: severity, Code: item.Code, Path: item.Path, Message: item.Message})
	}
	return out, nil
}
