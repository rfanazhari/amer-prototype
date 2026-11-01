package integration_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// assignmentCreateCase names one PostgreSQL assignment-create proof.
type assignmentCreateCase struct {
	name string
	run  func(t *testing.T, graph configHTTP)
}

// assignmentCreateView is the 201 pin payload.
type assignmentCreateView struct {
	ID                  string `json:"id"`
	CandidateID         string `json:"candidateId"`
	InstanceID          string `json:"instanceId"`
	PublishedRevisionID string `json:"publishedRevisionId"`
}

// TestInterviewAssignmentCreateHTTP proves T13 CORE-ACP-013 pin create on version-7 foreign keys.
func TestInterviewAssignmentCreateHTTP(t *testing.T) {
	tests := []assignmentCreateCase{
		{name: "hr create pins active revision and later publication keeps it", run: proveAssignmentCreateRetainsPin},
		{name: "admin is denied before insert", run: proveAssignmentCreateAdminDenied},
		{name: "unpublished instance is conflict", run: proveAssignmentCreateUnpublished},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, newConfigHTTP(t))
		})
	}
}

// proveAssignmentCreateRetainsPin creates through HTTP, invites with the assignment id, and keeps the pin after republication.
func proveAssignmentCreateRetainsPin(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	created := createInstance(t, graph, hr, "hr-interviewer", uniqueLabel("Assign "), `{}`)
	replaced := replaceInstance(t, graph, hr, created.ID, 1, hrConfiguration(acpHRTopic))
	validated := validateInstance(t, graph, hr, created.ID, replaced.DraftVersion)
	first := publishInstance(t, graph, hr, created.ID, replaced.DraftVersion, validated.ValidationID, commandKey(), http.StatusCreated)
	candidateID := createACPCandidate(t, graph, hr)
	assigned := createACPAssignment(t, graph, hr, candidateID, created.ID)
	require.Equal(t, first.RevisionID, assigned.PublishedRevisionID)
	require.Equal(t, candidateID, assigned.CandidateID)
	invited := doACP(t, graph.router, http.MethodPost, "/admin/interview-assignments/"+assigned.ID+"/invitations", `{}`, hr.csrf, "", hr.cookie)
	require.Equal(t, http.StatusCreated, invited.Code, invited.Body.String())
	missing := doACP(t, graph.router, http.MethodPost, "/admin/interview-assignments/"+candidateID+"/invitations", `{}`, hr.csrf, "", hr.cookie)
	requireCode(t, missing, http.StatusNotFound, errs.NotFound)
	replaceInstance(t, graph, hr, created.ID, replaced.DraftVersion, hrConfiguration(acpHREditedTopic))
	secondVal := validateInstance(t, graph, hr, created.ID, replaced.DraftVersion+1)
	second := publishInstance(t, graph, hr, created.ID, replaced.DraftVersion+1, secondVal.ValidationID, commandKey(), http.StatusCreated)
	require.NotEqual(t, first.RevisionID, second.RevisionID)
	require.Equal(t, first.RevisionID, pinnedRevision(t, graph, assigned.ID))
	later := createACPAssignment(t, graph, hr, candidateID, created.ID)
	require.Equal(t, second.RevisionID, later.PublishedRevisionID)
}

// proveAssignmentCreateAdminDenied refuses ADMIN before any assignment insert.
func proveAssignmentCreateAdminDenied(t *testing.T, graph configHTTP) {
	t.Helper()
	admin := loginAdminSession(t, graph)
	hr := provisionHR(t, graph)
	created := createInstance(t, graph, hr, "hr-interviewer", uniqueLabel("AdminPin "), `{}`)
	replaced := replaceInstance(t, graph, hr, created.ID, 1, hrConfiguration(acpHRTopic))
	validated := validateInstance(t, graph, hr, created.ID, replaced.DraftVersion)
	publishInstance(t, graph, hr, created.ID, replaced.DraftVersion, validated.ValidationID, commandKey(), http.StatusCreated)
	candidateID := createACPCandidate(t, graph, hr)
	denied := doACP(t, graph.router, http.MethodPost, "/admin/interview-assignments", assignmentCreateBody(candidateID, created.ID), admin.csrf, "", admin.cookie)
	requireCode(t, denied, http.StatusForbidden, errs.IAForbidden)
	require.Zero(t, assignmentCount(t, graph, candidateID))
}

// proveAssignmentCreateUnpublished rejects an HR instance without an active revision.
func proveAssignmentCreateUnpublished(t *testing.T, graph configHTTP) {
	t.Helper()
	hr := provisionHR(t, graph)
	created := createInstance(t, graph, hr, "hr-interviewer", uniqueLabel("DraftPin "), `{}`)
	candidateID := createACPCandidate(t, graph, hr)
	denied := doACP(t, graph.router, http.MethodPost, "/admin/interview-assignments", assignmentCreateBody(candidateID, created.ID), hr.csrf, "", hr.cookie)
	requireCode(t, denied, http.StatusConflict, errs.IANoActiveRevision)
	require.Zero(t, assignmentCount(t, graph, candidateID))
}

// createACPCandidate posts a candidate through the composed router.
func createACPCandidate(t *testing.T, graph configHTTP, session staffSession) string {
	t.Helper()
	email := "andi-" + uuid.NewString() + "@example.com"
	rec := doACP(t, graph.router, http.MethodPost, "/admin/candidates", `{"name":"Andi Pratama","email":"`+email+`"}`, session.csrf, "", session.cookie)
	view := acpData[struct {
		ID string `json:"id"`
	}](t, rec, http.StatusCreated)
	require.NotEmpty(t, view.ID)
	return view.ID
}

// createACPAssignment posts the approved assignment create body.
func createACPAssignment(t *testing.T, graph configHTTP, session staffSession, candidateID, instanceID string) assignmentCreateView {
	t.Helper()
	rec := doACP(t, graph.router, http.MethodPost, "/admin/interview-assignments", assignmentCreateBody(candidateID, instanceID), session.csrf, "", session.cookie)
	view := acpData[assignmentCreateView](t, rec, http.StatusCreated)
	require.NotEmpty(t, view.ID)
	require.Equal(t, candidateID, view.CandidateID)
	require.Equal(t, instanceID, view.InstanceID)
	require.NotContains(t, rec.Body.String(), `"createdAt"`)
	return view
}

// assignmentCreateBody builds the approved T12 request object.
func assignmentCreateBody(candidateID, instanceID string) string {
	return `{"candidateId":"` + candidateID + `","instanceId":"` + instanceID + `"}`
}

// pinnedRevision reads published_revision_id for one assignment.
func pinnedRevision(t *testing.T, graph configHTTP, assignmentID string) string {
	t.Helper()
	var revision string
	require.NoError(t, graph.db.Raw(`SELECT published_revision_id::text FROM interview_assignments WHERE id = ?`, assignmentID).Scan(&revision).Error)
	return revision
}

// assignmentCount counts pins for one candidate.
func assignmentCount(t *testing.T, graph configHTTP, candidateID string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, graph.db.Raw(`SELECT COUNT(*) FROM interview_assignments WHERE candidate_id = ?`, candidateID).Scan(&count).Error)
	return count
}
