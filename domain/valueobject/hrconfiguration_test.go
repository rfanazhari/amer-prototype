package valueobject_test

import (
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	jobDescriptionID = "11111111-1111-4111-8111-111111111111"
	salaryCeilingID  = "22222222-2222-4222-8222-222222222222"
)

// hrConstructionCase names a full-replacement draft and whether storage bounds reject it.
type hrConstructionCase struct {
	name    string
	input   valueobject.HRInput
	wantErr bool
}

// validHRInput returns the approved valid HR example as an editable replacement body.
func validHRInput() valueobject.HRInput {
	return valueobject.HRInput{
		JobDescriptionRef: &valueobject.RefInput{ID: jobDescriptionID, Version: "1"},
		MandatoryQuestions: []valueobject.QuestionInput{
			{ID: "background", Stage: "background", Text: "Describe your backend experience."},
			{ID: "screening", Stage: "screening", Text: "How do you protect a concurrent update?"},
			{ID: "compensation", Stage: "compensation", Text: "What are your compensation expectations?"},
		},
		CompetencyTopics: []string{"concurrency"},
		StageSeconds:     &valueobject.StageSecondsInput{Background: 60, Screening: 240, Compensation: 90, Closing: 30},
		CompensationPolicy: &valueobject.CompensationInput{
			DiscussionMode:   "expectations_only",
			SalaryCeilingRef: &valueobject.RefInput{ID: salaryCeilingID, Version: "1"},
		},
	}
}

// mustHR constructs a storage-valid HR draft or fails the test.
func mustHR(t *testing.T, in valueobject.HRInput) valueobject.HRConfiguration {
	t.Helper()
	cfg, err := valueobject.NewHRConfiguration(in)
	require.NoError(t, err)
	return cfg
}

// TestNewHRConfigurationAcceptsIncompleteDraftsAndRejectsBounds covers the supported field/boundary matrix.
func TestNewHRConfigurationAcceptsIncompleteDraftsAndRejectsBounds(t *testing.T) {
	overlongQuestion := validHRInput()
	overlongQuestion.MandatoryQuestions[0].Text = strings.Repeat("问", 2001)
	overlongTopic := validHRInput()
	overlongTopic.CompetencyTopics[0] = strings.Repeat("题", 121)
	duplicateTopic := validHRInput()
	duplicateTopic.CompetencyTopics = []string{"concurrency", " concurrency "}
	duplicateQuestion := validHRInput()
	duplicateQuestion.MandatoryQuestions[1].ID = "background"
	unknownStage := validHRInput()
	unknownStage.MandatoryQuestions[0].Stage = "followup"
	badQuestionID := validHRInput()
	badQuestionID.MandatoryQuestions[0].ID = "1background"
	controlText := validHRInput()
	controlText.MandatoryQuestions[0].Text = "hello\x01world"
	tooManyQuestions := validHRInput()
	tooManyQuestions.MandatoryQuestions = make([]valueobject.QuestionInput, 31)
	for i := range tooManyQuestions.MandatoryQuestions {
		tooManyQuestions.MandatoryQuestions[i] = valueobject.QuestionInput{ID: "q" + strings.Repeat("a", i%10), Stage: "background", Text: "ok"}
	}
	tooManyQuestions.MandatoryQuestions[10].ID = "qabcdefghij"
	tooManyTopics := validHRInput()
	tooManyTopics.CompetencyTopics = make([]string, 21)
	for i := range tooManyTopics.CompetencyTopics {
		tooManyTopics.CompetencyTopics[i] = "topic-" + string(rune('a'+i))
	}
	negativeStage := validHRInput()
	negativeStage.StageSeconds = &valueobject.StageSecondsInput{Background: -1, Screening: 240, Compensation: 90, Closing: 30}
	overStage := validHRInput()
	overStage.StageSeconds = &valueobject.StageSecondsInput{Background: 901, Screening: 240, Compensation: 90, Closing: 30}
	badMode := validHRInput()
	badMode.CompensationPolicy.DiscussionMode = "negotiate"
	tests := []hrConstructionCase{
		{name: "valid example", input: validHRInput()},
		{name: "empty draft", input: valueobject.HRInput{}},
		{name: "nil slices", input: valueobject.HRInput{MandatoryQuestions: nil, CompetencyTopics: nil}},
		{name: "empty question text", input: func() valueobject.HRInput {
			in := validHRInput()
			in.MandatoryQuestions[0].Text = "  \n"
			return in
		}()},
		{name: "zero stage seconds", input: func() valueobject.HRInput {
			in := validHRInput()
			in.StageSeconds.Screening = 0
			return in
		}()},
		{name: "overlong question", input: overlongQuestion, wantErr: true},
		{name: "overlong topic", input: overlongTopic, wantErr: true},
		{name: "duplicate topic", input: duplicateTopic, wantErr: true},
		{name: "duplicate question id", input: duplicateQuestion, wantErr: true},
		{name: "unknown stage", input: unknownStage, wantErr: true},
		{name: "invalid question id", input: badQuestionID, wantErr: true},
		{name: "control character", input: controlText, wantErr: true},
		{name: "too many questions", input: tooManyQuestions, wantErr: true},
		{name: "too many topics", input: tooManyTopics, wantErr: true},
		{name: "negative stage seconds", input: negativeStage, wantErr: true},
		{name: "stage seconds above 900", input: overStage, wantErr: true},
		{name: "unknown discussion mode", input: badMode, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valueobject.NewHRConfiguration(tt.input)
			if tt.wantErr {
				require.ErrorIs(t, err, valueobject.ErrInvalidConfiguration)
				assert.Equal(t, valueobject.HRConfiguration{}, got)
				return
			}
			require.NoError(t, err)
			if tt.name == "empty draft" || tt.name == "nil slices" {
				assert.Nil(t, got.JobDescriptionRef())
				assert.Empty(t, got.MandatoryQuestions())
				assert.Empty(t, got.CompetencyTopics())
				assert.Nil(t, got.StageSeconds())
				assert.Nil(t, got.CompensationPolicy())
				return
			}
			require.NotNil(t, got.JobDescriptionRef())
			assert.Equal(t, jobDescriptionID, got.JobDescriptionRef().ID())
			assert.Equal(t, "1", got.JobDescriptionRef().Version())
			require.Len(t, got.MandatoryQuestions(), 3)
			assert.Equal(t, enum.StageBackground, got.MandatoryQuestions()[0].Stage())
			if tt.name == "empty question text" {
				assert.Empty(t, got.MandatoryQuestions()[0].Text())
			} else {
				assert.Equal(t, "Describe your backend experience.", got.MandatoryQuestions()[0].Text())
			}
			assert.Equal(t, []string{"concurrency"}, got.CompetencyTopics())
			require.NotNil(t, got.StageSeconds())
			if tt.name == "zero stage seconds" {
				assert.Equal(t, 0, got.StageSeconds().Screening())
			} else {
				assert.Equal(t, 240, got.StageSeconds().Screening())
			}
			require.NotNil(t, got.CompensationPolicy())
			assert.Equal(t, "expectations_only", got.CompensationPolicy().DiscussionMode())
			require.NotNil(t, got.CompensationPolicy().SalaryCeilingRef())
			assert.Equal(t, salaryCeilingID, got.CompensationPolicy().SalaryCeilingRef().ID())
		})
	}
}

// TestHRConfigurationNormalizesTextAndOrdersTopics without collapsing internal whitespace.
func TestHRConfigurationNormalizesTextAndOrdersTopics(t *testing.T) {
	in := validHRInput()
	in.MandatoryQuestions[0].Text = "  line\r\nbreak  "
	in.CompetencyTopics = []string{"zeta", "Alpha", "beta"}
	got := mustHR(t, in)
	require.NotEmpty(t, got.MandatoryQuestions())
	assert.Equal(t, "line\nbreak", got.MandatoryQuestions()[0].Text())
	assert.Equal(t, []string{"Alpha", "beta", "zeta"}, got.CompetencyTopics())
	assert.Equal(t, "background", got.MandatoryQuestions()[0].ID())
	assert.Equal(t, "screening", got.MandatoryQuestions()[1].ID())
}

// TestHRConfigurationCopyIsolation keeps nested slices and pointers independently owned.
func TestHRConfigurationCopyIsolation(t *testing.T) {
	in := validHRInput()
	cfg := mustHR(t, in)
	in.CompetencyTopics[0] = "mutated"
	in.MandatoryQuestions[0].Text = "mutated"
	in.JobDescriptionRef.ID = "33333333-3333-4333-8333-333333333333"
	require.NotEmpty(t, cfg.CompetencyTopics())
	require.NotEmpty(t, cfg.MandatoryQuestions())
	require.NotNil(t, cfg.JobDescriptionRef())
	assert.Equal(t, "concurrency", cfg.CompetencyTopics()[0])
	assert.Equal(t, "Describe your backend experience.", cfg.MandatoryQuestions()[0].Text())
	assert.Equal(t, jobDescriptionID, cfg.JobDescriptionRef().ID())
	topics := cfg.CompetencyTopics()
	topics[0] = "caller"
	assert.Equal(t, "concurrency", cfg.CompetencyTopics()[0])
	clone := cfg.Clone()
	assert.Equal(t, cfg.CompetencyTopics(), clone.CompetencyTopics())
	require.NotNil(t, clone.JobDescriptionRef())
	require.NotNil(t, clone.StageSeconds())
	require.NotNil(t, clone.CompensationPolicy())
	require.NotNil(t, clone.CompensationPolicy().SalaryCeilingRef())
	require.NotSame(t, cfg.JobDescriptionRef(), clone.JobDescriptionRef())
	require.NotSame(t, cfg.StageSeconds(), clone.StageSeconds())
	require.NotSame(t, cfg.CompensationPolicy().SalaryCeilingRef(), clone.CompensationPolicy().SalaryCeilingRef())
}
