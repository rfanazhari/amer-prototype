package agentconfig

import (
	"fmt"

	aggregate "github.com/rfanazhari/amer-prototype/domain/aggregate/agentconfig"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

const (
	questionCapacitySeconds = 30
	followupSpareSeconds    = 30
	minPublicationTotal     = 300
	maxPublicationTotal     = 600
	demoJobRole             = "Backend Engineer"
)

// ReferenceReadiness is a detached exact-version reference fact, not a live provider check.
type ReferenceReadiness struct {
	Found, Ready      bool
	ID, Version, Role string
}

// HRReferenceContext holds locally resolved job-description and compensation facts.
type HRReferenceContext struct {
	JobDescription, SalaryCeiling ReferenceReadiness
}

// AssessHR records deterministic HR findings without model or provider calls.
func AssessHR(cfg valueobject.HRConfiguration, refs HRReferenceContext) []aggregate.Finding {
	findings := make([]aggregate.Finding, 0)
	findings = append(findings, assessJobDescription(cfg.JobDescriptionRef(), refs.JobDescription)...)
	findings = append(findings, assessQuestions(cfg.MandatoryQuestions())...)
	findings = append(findings, assessTopics(cfg.CompetencyTopics())...)
	findings = append(findings, assessStageSeconds(cfg.StageSeconds(), cfg.MandatoryQuestions())...)
	findings = append(findings, assessCompensation(cfg.CompensationPolicy(), refs.SalaryCeiling)...)
	sortFindings(findings)
	if len(findings) == 0 {
		return nil
	}
	return findings
}

// assessJobDescription requires a locally ready Backend Engineer job description.
func assessJobDescription(ref *valueobject.StableRef, readiness ReferenceReadiness) []aggregate.Finding {
	if ref == nil {
		return []aggregate.Finding{errorFinding(CodeRequiredValue, "/jobDescriptionRef", messageRequiredValue)}
	}
	if !readiness.Found || !readiness.Ready || readiness.Role != demoJobRole {
		return []aggregate.Finding{errorFinding(CodeReferenceNotReady, "/jobDescriptionRef", messageReferenceNotReady)}
	}
	return nil
}

// assessQuestions requires nonempty text in background, screening and compensation.
func assessQuestions(questions []valueobject.MandatoryQuestion) []aggregate.Finding {
	findings := make([]aggregate.Finding, 0)
	nonempty := map[enum.QuestionStage]bool{}
	for i, question := range questions {
		if question.Text() == "" {
			findings = append(findings, errorFinding(CodeQuestionTextRequired, fmt.Sprintf("/mandatoryQuestions/%d/text", i), messageQuestionTextRequired))
			continue
		}
		nonempty[question.Stage()] = true
	}
	for _, stage := range enum.RequiredPublicationStages() {
		if !nonempty[stage] {
			findings = append(findings, errorFinding(CodeQuestionStageRequired, "/mandatoryQuestions", messageQuestionStageRequired))
			break
		}
	}
	return findings
}

// assessTopics requires at least one nonempty competency topic.
func assessTopics(topics []string) []aggregate.Finding {
	for _, topic := range topics {
		if topic != "" {
			return nil
		}
	}
	return []aggregate.Finding{errorFinding(CodeCompetencyRequired, "/competencyTopics", messageCompetencyRequired)}
}

// assessStageSeconds enforces positive durations, the 300–600 total, question capacity and screening spare.
func assessStageSeconds(stages *valueobject.StageSeconds, questions []valueobject.MandatoryQuestion) []aggregate.Finding {
	if stages == nil {
		return []aggregate.Finding{errorFinding(CodeRequiredValue, "/stageSeconds", messageRequiredValue)}
	}
	findings := make([]aggregate.Finding, 0)
	seconds := map[enum.QuestionStage]int{
		enum.StageBackground:   stages.Background(),
		enum.StageScreening:    stages.Screening(),
		enum.StageCompensation: stages.Compensation(),
		enum.StageClosing:      stages.Closing(),
	}
	total := 0
	for _, stage := range []enum.QuestionStage{enum.StageBackground, enum.StageScreening, enum.StageCompensation, enum.StageClosing} {
		value := seconds[stage]
		total += value
		if value <= 0 {
			findings = append(findings, errorFinding(CodeStageBudgetInvalid, "/stageSeconds/"+stage.String(), messageStageBudgetInvalid))
		}
	}
	if total < minPublicationTotal || total > maxPublicationTotal {
		findings = append(findings, errorFinding(CodeStageBudgetInvalid, "/stageSeconds", messageStageBudgetInvalid))
	}
	counts := map[enum.QuestionStage]int{}
	for _, question := range questions {
		counts[question.Stage()]++
	}
	for _, stage := range []enum.QuestionStage{enum.StageBackground, enum.StageScreening, enum.StageCompensation, enum.StageClosing} {
		needed := counts[stage] * questionCapacitySeconds
		if needed > seconds[stage] {
			findings = append(findings, errorFinding(CodeQuestionBudgetExceeded, "/stageSeconds/"+stage.String(), messageQuestionBudgetExceeded))
		}
	}
	if seconds[enum.StageScreening]-counts[enum.StageScreening]*questionCapacitySeconds < followupSpareSeconds {
		findings = append(findings, errorFinding(CodeFollowupBudgetRequired, "/stageSeconds/screening", messageFollowupBudgetRequired))
	}
	return findings
}

// assessCompensation requires a locally ready confidential ceiling reference.
func assessCompensation(policy *valueobject.CompensationPolicy, readiness ReferenceReadiness) []aggregate.Finding {
	if policy == nil {
		return []aggregate.Finding{errorFinding(CodeRequiredValue, "/compensationPolicy", messageRequiredValue)}
	}
	if policy.SalaryCeilingRef() == nil {
		return []aggregate.Finding{errorFinding(CodeRequiredValue, "/compensationPolicy/salaryCeilingRef", messageRequiredValue)}
	}
	if !readiness.Found || !readiness.Ready {
		return []aggregate.Finding{errorFinding(CodeReferenceNotReady, "/compensationPolicy/salaryCeilingRef", messageReferenceNotReady)}
	}
	return nil
}
