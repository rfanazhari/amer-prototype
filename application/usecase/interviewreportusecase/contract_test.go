package interviewreportusecase_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/application/query/interviewreportquery"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/interviewreportusecase"
	"github.com/rfanazhari/amer-prototype/test/mocks"
)

// TestGeneratedMocksSatisfyReportContracts keeps the generated mocks aligned with the report ports and operations.
func TestGeneratedMocksSatisfyReportContracts(t *testing.T) {
	var (
		_ service.InterviewReportAuthority           = (*mocks.InterviewReportAuthority)(nil)
		_ interviewreportport.Evidence               = (*mocks.InterviewReportEvidence)(nil)
		_ interviewreportport.Writer                 = (*mocks.InterviewReportWriter)(nil)
		_ interviewreportusecase.AdmitOperation      = (*mocks.InterviewReportAdmitOperation)(nil)
		_ interviewreportusecase.RegenerateOperation = (*mocks.InterviewReportRegenerateOperation)(nil)
		_ interviewreportquery.ReadOperation         = (*mocks.InterviewReportReadOperation)(nil)
	)
}
