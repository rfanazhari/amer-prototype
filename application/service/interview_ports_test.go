package service_test

import (
	"reflect"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInterviewPortOperations specifies the callable boundaries before any adapter is implemented.
func TestInterviewPortOperations(t *testing.T) {
	// portCase identifies one required interface and its operations.
	type portCase struct {
		name    string
		port    reflect.Type
		methods []string
	}
	tests := []portCase{
		{name: "lifecycle", port: reflect.TypeFor[service.InterviewLifecycle](), methods: []string{"Activate", "Fence", "WithProgress", "Complete", "Fail"}},
		{name: "driver", port: reflect.TypeFor[service.InterviewDriver](), methods: []string{"Create", "Attach", "Observe", "SupplyContext", "Control", "Close"}},
		{name: "progress", port: reflect.TypeFor[interviewport.ProgressStore](), methods: []string{"Insert", "Find", "Update"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, len(tt.methods), tt.port.NumMethod())
			for _, name := range tt.methods {
				method, ok := tt.port.MethodByName(name)
				require.True(t, ok, "missing operation %s", name)
				assert.Equal(t, "context.Context", method.Type.In(0).String())
				assert.Equal(t, "error", method.Type.Out(method.Type.NumOut()-1).String())
			}
		})
	}
}
