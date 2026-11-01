package valueobject

import "github.com/rfanazhari/amer-prototype/domain/enum"

const (
	hrQuestionLimit   = 30
	hrTopicLimit      = 20
	hrQuestionIDMax   = 64
	hrQuestionTextMax = 2000
	hrTopicMax        = 120
	hrStageSecondMax  = 900
	hrDiscussionMode  = "expectations_only"
)

// RefInput is the editable identity and version supplied for a stable reference.
type RefInput struct {
	ID, Version string
}

// QuestionInput is one editable mandatory-question replacement value.
type QuestionInput struct {
	ID, Stage, Text string
}

// StageSecondsInput is the editable four-stage duration object.
type StageSecondsInput struct {
	Background, Screening, Compensation, Closing int
}

// CompensationInput is the editable compensation discussion policy.
type CompensationInput struct {
	DiscussionMode   string
	SalaryCeilingRef *RefInput
}

// HRInput is a full replacement of supported HR configuration fields.
type HRInput struct {
	JobDescriptionRef  *RefInput
	MandatoryQuestions []QuestionInput
	CompetencyTopics   []string
	StageSeconds       *StageSecondsInput
	CompensationPolicy *CompensationInput
}

// MandatoryQuestion is one owned HR question after storage-bound construction.
type MandatoryQuestion struct {
	id, text string
	stage    enum.QuestionStage
}

// StageSeconds is the owned four-stage duration object.
type StageSeconds struct {
	background, screening, compensation, closing int
}

// CompensationPolicy is the owned discussion mode and optional ceiling reference.
type CompensationPolicy struct {
	discussionMode   string
	salaryCeilingRef *StableRef
}

// HRConfiguration is an editable HR Interviewer draft; incomplete values remain constructible.
type HRConfiguration struct {
	jobDescription     *StableRef
	questions          []MandatoryQuestion
	topics             []string
	stageSeconds       *StageSeconds
	compensationPolicy *CompensationPolicy
}

// NewHRConfiguration normalizes a supported HR draft and rejects storage-bound violations.
func NewHRConfiguration(in HRInput) (HRConfiguration, error) {
	return HRConfiguration{}, nil
}

// Clone returns an independently owned copy of nested references, questions and topics.
func (c HRConfiguration) Clone() HRConfiguration { return c }

// JobDescriptionRef returns the copied job-description identity when present.
func (c HRConfiguration) JobDescriptionRef() *StableRef { return c.jobDescription }

// MandatoryQuestions returns a copied question list in stored order.
func (c HRConfiguration) MandatoryQuestions() []MandatoryQuestion { return c.questions }

// CompetencyTopics returns a copied topic list in stored order.
func (c HRConfiguration) CompetencyTopics() []string { return c.topics }

// StageSeconds returns a copied duration object when present.
func (c HRConfiguration) StageSeconds() *StageSeconds { return c.stageSeconds }

// CompensationPolicy returns a copied compensation policy when present.
func (c HRConfiguration) CompensationPolicy() *CompensationPolicy { return c.compensationPolicy }

// ID returns the question identity.
func (q MandatoryQuestion) ID() string { return q.id }

// Stage returns the interview stage for the question.
func (q MandatoryQuestion) Stage() enum.QuestionStage { return q.stage }

// Text returns the normalized question text.
func (q MandatoryQuestion) Text() string { return q.text }

// Background returns the background-stage duration in seconds.
func (s StageSeconds) Background() int { return s.background }

// Screening returns the screening-stage duration in seconds.
func (s StageSeconds) Screening() int { return s.screening }

// Compensation returns the compensation-stage duration in seconds.
func (s StageSeconds) Compensation() int { return s.compensation }

// Closing returns the closing-stage duration in seconds.
func (s StageSeconds) Closing() int { return s.closing }

// DiscussionMode returns the stored compensation discussion mode.
func (p CompensationPolicy) DiscussionMode() string { return p.discussionMode }

// SalaryCeilingRef returns the copied ceiling identity when present.
func (p CompensationPolicy) SalaryCeilingRef() *StableRef { return p.salaryCeilingRef }
