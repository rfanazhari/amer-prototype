package interviewreportrepository

import (
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
)

// jobKind is the only outbox job kind this version stores.
const jobKind = "interview_report"

// toSnapshot restores the aggregate from stored rows and rejects rows that break its invariants.
// Statements must already be grouped by generation id in ordinal order.
func toSnapshot(report ReportModel, generations []GenerationModel, statements []StatementModel) (interviewreport.Snapshot, error) {
	grouped := make(map[string][]interviewreport.Statement, len(generations))
	for _, row := range statements {
		statement, err := interviewreport.RestoreStatement(toStatementInput(row))
		if err != nil {
			return interviewreport.Snapshot{}, err
		}
		grouped[row.GenerationID] = append(grouped[row.GenerationID], statement)
	}
	snap := interviewreport.Snapshot{
		ID: report.ID, ExecutionID: report.ExecutionID, AssignmentID: report.AssignmentID, CreatedAt: report.CreatedAt.UTC(),
	}
	for _, row := range generations {
		generation := interviewreport.Generation{
			ID: row.ID, Number: row.GenerationNumber, State: interviewreport.State(row.State),
			CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(), Statements: grouped[row.ID],
		}
		if row.FailureReason != nil {
			generation.FailureReason = *row.FailureReason
		}
		snap.Generations = append(snap.Generations, generation)
	}
	restored, err := interviewreport.RestoreReport(snap)
	if err != nil {
		return interviewreport.Snapshot{}, err
	}
	return restored.Snapshot(), nil
}

// toStatementInput maps a stored statement to the domain restoration input.
func toStatementInput(row StatementModel) interviewreport.StatementInput {
	input := interviewreport.StatementInput{
		Kind: interviewreport.Kind(row.Kind), Body: row.Body, PresentsAsAnswer: row.PresentsAsAnswer,
	}
	if row.EvidenceEntryID != nil || row.EvidenceRevision != nil {
		citation := interviewreport.Citation{}
		if row.EvidenceEntryID != nil {
			citation.EntryID = *row.EvidenceEntryID
		}
		if row.EvidenceRevision != nil {
			citation.Revision = *row.EvidenceRevision
		}
		input.Citation = &citation
	}
	return input
}

// fromAdmission maps an admission to the four rows it inserts. The caller has checked that the report
// holds exactly one pending generation. The job starts unclaimed with fencing token 1 and no window.
func fromAdmission(a interviewreportport.Admission) (ReportModel, GenerationModel, JobModel, OutboxModel) {
	generation := a.Report.Generations[0]
	created := generation.CreatedAt.UTC()
	return ReportModel{
			ID: a.Report.ID, ExecutionID: a.Report.ExecutionID, AssignmentID: a.Report.AssignmentID, CreatedAt: a.Report.CreatedAt.UTC(),
		}, GenerationModel{
			ID: generation.ID, ReportID: a.Report.ID, GenerationNumber: generation.Number, State: string(generation.State),
			CreatedAt: created, UpdatedAt: generation.UpdatedAt.UTC(),
		}, JobModel{
			ID: a.JobID, GenerationID: generation.ID, State: "pending", AttemptCount: 0, FencingToken: 1,
			CreatedAt: created, WaitDeadlineAt: a.WaitDeadlineAt.UTC(), EligibleAt: created, UpdatedAt: created,
		}, OutboxModel{ID: a.OutboxID, JobKind: jobKind, JobID: a.JobID, CreatedAt: created}
}
