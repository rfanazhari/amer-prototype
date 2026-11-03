package interviewreporthandler

import "github.com/rfanazhari/amer-prototype/application/dto"

// generationView is the complete generation wire contract including nullable failure reason.
type generationView struct {
	ID            string  `json:"id"`
	Number        int     `json:"number"`
	State         string  `json:"state"`
	FailureReason *string `json:"failureReason"`
	CreatedAt     string  `json:"createdAt"`
}

// observationView always cites finalized evidence for statements presented as answers.
type observationView struct {
	Body             string `json:"body"`
	EvidenceEntryID  string `json:"evidenceEntryId"`
	EvidenceRevision int    `json:"evidenceRevision"`
}

// gapView carries coverage identity without any evidence citation.
type gapView struct {
	Body       string  `json:"body"`
	QuestionID *string `json:"questionId"`
}

// uncertaintyView omits absent citations and never presents itself as an answer.
type uncertaintyView struct {
	Body             string  `json:"body"`
	EvidenceEntryID  *string `json:"evidenceEntryId,omitempty"`
	EvidenceRevision *int    `json:"evidenceRevision,omitempty"`
}

// accountView exposes only the approved account of a succeeded generation.
type accountView struct {
	GenerationID  string            `json:"generationId"`
	Observations  []observationView `json:"observations"`
	Gaps          []gapView         `json:"gaps"`
	Uncertainties []uncertaintyView `json:"uncertainties"`
}

// reportView pairs the newest generation with the previous succeeded account when available.
type reportView struct {
	AssignmentID string         `json:"assignmentId"`
	ExecutionID  string         `json:"executionId"`
	Generation   generationView `json:"generation"`
	Report       *accountView   `json:"report"`
}

// encodeGeneration normalizes the public instant to UTC milliseconds.
func encodeGeneration(g dto.ReportGeneration) generationView {
	return generationView{ID: g.ID, Number: g.Number, State: g.State, FailureReason: g.FailureReason, CreatedAt: g.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z")}
}

// encodeView explicitly maps detached DTOs without exposing internal or provider fields.
func encodeView(v dto.InterviewReportView) reportView {
	out := reportView{AssignmentID: v.AssignmentID, ExecutionID: v.ExecutionID, Generation: encodeGeneration(v.Generation)}
	if v.Report == nil {
		return out
	}
	a := v.Report
	out.Report = &accountView{GenerationID: a.GenerationID, Observations: []observationView{}, Gaps: []gapView{}, Uncertainties: []uncertaintyView{}}
	for _, s := range a.Observations {
		out.Report.Observations = append(out.Report.Observations, observationView{Body: s.Body, EvidenceEntryID: s.EvidenceEntryID, EvidenceRevision: s.EvidenceRevision})
	}
	for _, s := range a.Gaps {
		out.Report.Gaps = append(out.Report.Gaps, gapView{Body: s.Body, QuestionID: s.QuestionID})
	}
	for _, s := range a.Uncertainties {
		out.Report.Uncertainties = append(out.Report.Uncertainties, uncertaintyView{Body: s.Body, EvidenceEntryID: s.EvidenceEntryID, EvidenceRevision: s.EvidenceRevision})
	}
	return out
}
