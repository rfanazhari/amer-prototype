package service_test

import (
	"testing"
	"time"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/stretchr/testify/assert"
)

// TestDefaultInterviewBudgets pins the independent platform limits passed through the runtime ports.
func TestDefaultInterviewBudgets(t *testing.T) {
	// budgetCase describes one externally specified limit.
	type budgetCase struct {
		name     string
		expected any
		actual   any
	}
	b := service.DefaultInterviewBudgets()
	tests := []budgetCase{
		{name: "activation attempts", expected: 2, actual: b.ActivationAttempts},
		{name: "activation attempt timeout", expected: 30 * time.Second, actual: b.ActivationAttemptTimeout},
		{name: "activation window", expected: 90 * time.Second, actual: b.ActivationWindow},
		{name: "reconnect window", expected: 120 * time.Second, actual: b.ReconnectWindow},
		{name: "reconnect attempts per episode", expected: 2, actual: b.ReconnectAttempts},
		{name: "reconnect episodes", expected: 2, actual: b.ReconnectEpisodes},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { assert.Equal(t, tt.expected, tt.actual) })
	}
}
