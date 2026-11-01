package enum

// QuestionStage is an HR interview stage stored on a mandatory question.
type QuestionStage string

const (
	StageBackground   QuestionStage = "background"
	StageScreening    QuestionStage = "screening"
	StageCompensation QuestionStage = "compensation"
	StageClosing      QuestionStage = "closing"
)

// Valid reports whether the stage is one of the four supported interview stages.
func (s QuestionStage) Valid() bool {
	return s == StageBackground || s == StageScreening || s == StageCompensation || s == StageClosing
}

// String returns the stable storage and API representation of the stage.
func (s QuestionStage) String() string { return string(s) }

// RequiredPublicationStages returns the stages that need at least one nonempty question.
func RequiredPublicationStages() []QuestionStage {
	return []QuestionStage{StageBackground, StageScreening, StageCompensation}
}
