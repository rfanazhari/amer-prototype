package valueobject

import (
	"sort"

	"github.com/rfanazhari/amer-prototype/domain/enum"
)

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
	job, err := optionalRef(in.JobDescriptionRef)
	if err != nil {
		return HRConfiguration{}, err
	}
	questions, err := normalizeQuestions(in.MandatoryQuestions)
	if err != nil {
		return HRConfiguration{}, err
	}
	topics, err := normalizeTopics(in.CompetencyTopics)
	if err != nil {
		return HRConfiguration{}, err
	}
	stages, err := normalizeStageSeconds(in.StageSeconds)
	if err != nil {
		return HRConfiguration{}, err
	}
	policy, err := normalizeCompensation(in.CompensationPolicy)
	if err != nil {
		return HRConfiguration{}, err
	}
	return HRConfiguration{jobDescription: job, questions: questions, topics: topics, stageSeconds: stages, compensationPolicy: policy}, nil
}

// optionalRef constructs a stable reference when the caller supplied one.
func optionalRef(in *RefInput) (*StableRef, error) {
	if in == nil {
		return nil, nil
	}
	ref, err := NewStableRef(in.ID, in.Version)
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

// normalizeQuestions keeps question order, unique identities and storage-bounded text.
func normalizeQuestions(in []QuestionInput) ([]MandatoryQuestion, error) {
	if len(in) > hrQuestionLimit {
		return nil, ErrInvalidConfiguration
	}
	out := make([]MandatoryQuestion, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, item := range in {
		if !validQuestionID(item.ID) {
			return nil, ErrInvalidConfiguration
		}
		if _, exists := seen[item.ID]; exists {
			return nil, ErrInvalidConfiguration
		}
		stage := enum.QuestionStage(item.Stage)
		if !stage.Valid() {
			return nil, ErrInvalidConfiguration
		}
		text, err := normalizeEditableText(item.Text, hrQuestionTextMax)
		if err != nil {
			return nil, err
		}
		seen[item.ID] = struct{}{}
		out = append(out, MandatoryQuestion{id: item.ID, stage: stage, text: text})
	}
	return out, nil
}

// validQuestionID reports an identity matching [a-z][a-z0-9_-]{0,63}.
func validQuestionID(id string) bool {
	if len(id) < 1 || len(id) > hrQuestionIDMax || id[0] < 'a' || id[0] > 'z' {
		return false
	}
	for i := 1; i < len(id); i++ {
		c := id[i]
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

// normalizeTopics rejects duplicates after normalization and stores lexical order.
func normalizeTopics(in []string) ([]string, error) {
	if len(in) > hrTopicLimit {
		return nil, ErrInvalidConfiguration
	}
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, item := range in {
		topic, err := normalizeEditableText(item, hrTopicMax)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[topic]; exists {
			return nil, ErrInvalidConfiguration
		}
		seen[topic] = struct{}{}
		out = append(out, topic)
	}
	sort.Strings(out)
	return out, nil
}

// normalizeStageSeconds accepts a four-stage object with each duration in 0–900 seconds.
func normalizeStageSeconds(in *StageSecondsInput) (*StageSeconds, error) {
	if in == nil {
		return nil, nil
	}
	if invalidStageSecond(in.Background) || invalidStageSecond(in.Screening) || invalidStageSecond(in.Compensation) || invalidStageSecond(in.Closing) {
		return nil, ErrInvalidConfiguration
	}
	return &StageSeconds{background: in.Background, screening: in.Screening, compensation: in.Compensation, closing: in.Closing}, nil
}

// invalidStageSecond reports a duration outside the storage bound.
func invalidStageSecond(value int) bool { return value < 0 || value > hrStageSecondMax }

// normalizeCompensation requires the approved discussion mode when a policy object is present.
func normalizeCompensation(in *CompensationInput) (*CompensationPolicy, error) {
	if in == nil {
		return nil, nil
	}
	if in.DiscussionMode != hrDiscussionMode {
		return nil, ErrInvalidConfiguration
	}
	ceiling, err := optionalRef(in.SalaryCeilingRef)
	if err != nil {
		return nil, err
	}
	return &CompensationPolicy{discussionMode: in.DiscussionMode, salaryCeilingRef: ceiling}, nil
}

// Clone returns an independently owned copy of nested references, questions and topics.
func (c HRConfiguration) Clone() HRConfiguration {
	return HRConfiguration{
		jobDescription:     copyRef(c.jobDescription),
		questions:          append([]MandatoryQuestion(nil), c.questions...),
		topics:             append([]string(nil), c.topics...),
		stageSeconds:       copyStageSeconds(c.stageSeconds),
		compensationPolicy: copyCompensation(c.compensationPolicy),
	}
}

// copyStageSeconds returns an independently owned duration object.
func copyStageSeconds(s *StageSeconds) *StageSeconds {
	if s == nil {
		return nil
	}
	copied := *s
	return &copied
}

// copyCompensation returns an independently owned policy including its ceiling reference.
func copyCompensation(p *CompensationPolicy) *CompensationPolicy {
	if p == nil {
		return nil
	}
	return &CompensationPolicy{discussionMode: p.discussionMode, salaryCeilingRef: copyRef(p.salaryCeilingRef)}
}

// JobDescriptionRef returns the copied job-description identity when present.
func (c HRConfiguration) JobDescriptionRef() *StableRef { return copyRef(c.jobDescription) }

// MandatoryQuestions returns a copied question list in stored order.
func (c HRConfiguration) MandatoryQuestions() []MandatoryQuestion {
	return append([]MandatoryQuestion(nil), c.questions...)
}

// CompetencyTopics returns a copied topic list in stored order.
func (c HRConfiguration) CompetencyTopics() []string { return append([]string(nil), c.topics...) }

// StageSeconds returns a copied duration object when present.
func (c HRConfiguration) StageSeconds() *StageSeconds { return copyStageSeconds(c.stageSeconds) }

// CompensationPolicy returns a copied compensation policy when present.
func (c HRConfiguration) CompensationPolicy() *CompensationPolicy {
	return copyCompensation(c.compensationPolicy)
}

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
func (p CompensationPolicy) SalaryCeilingRef() *StableRef { return copyRef(p.salaryCeilingRef) }
