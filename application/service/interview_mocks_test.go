package service_test

import (
	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/test/mocks"
)

// Generated mocks must retain every callable contract as the interfaces evolve.
var (
	_ service.InterviewLifecycle  = (*mocks.InterviewLifecycle)(nil)
	_ service.InterviewDriver     = (*mocks.InterviewDriver)(nil)
	_ interviewport.ProgressStore = (*mocks.ProgressStore)(nil)
)
