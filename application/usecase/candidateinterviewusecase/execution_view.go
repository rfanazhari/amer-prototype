package candidateinterviewusecase

import (
	"time"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Execution states reported by Execution Lifecycle that still accept a controller.
const (
	stateCreated = "CREATED"
	stateRunning = "RUNNING"
)

// live reports whether the execution state is non-terminal.
func live(state string) bool { return state == stateCreated || state == stateRunning }

// executionView builds the candidate-safe execution with a detached deadline copy and the controller facts.
func executionView(facts service.ExecutionFacts, isController bool, controllerVersion int64) dto.InterviewExecutionView {
	var deadline *time.Time
	if facts.HardDeadline != nil {
		c := *facts.HardDeadline
		deadline = &c
	}
	return dto.InterviewExecutionView{
		ID: facts.ID, State: facts.State, Version: facts.Version, HardDeadline: deadline,
		IsCurrentController: isController, ControllerVersion: controllerVersion,
	}
}
