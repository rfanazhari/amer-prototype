package enum_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/stretchr/testify/require"
)

// questionStageCase names a stage token and whether it is a supported interview stage.
type questionStageCase struct {
	name      string
	stage     enum.QuestionStage
	wantValid bool
}

// TestQuestionStageAcceptsOnlyCatalogStages rejects unknown and case-variant stages.
func TestQuestionStageAcceptsOnlyCatalogStages(t *testing.T) {
	tests := []questionStageCase{
		{name: "background", stage: enum.StageBackground, wantValid: true},
		{name: "screening", stage: enum.StageScreening, wantValid: true},
		{name: "compensation", stage: enum.StageCompensation, wantValid: true},
		{name: "closing", stage: enum.StageClosing, wantValid: true},
		{name: "empty", stage: "", wantValid: false},
		{name: "followup", stage: enum.QuestionStage("followup"), wantValid: false},
		{name: "uppercase", stage: enum.QuestionStage("BACKGROUND"), wantValid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantValid, tt.stage.Valid())
			if tt.wantValid {
				require.Equal(t, string(tt.stage), tt.stage.String())
			}
		})
	}
	require.Equal(t, []enum.QuestionStage{enum.StageBackground, enum.StageScreening, enum.StageCompensation}, enum.RequiredPublicationStages())
}
