package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
)

// candidateIntegrationCase names one real PostgreSQL candidate-access proof.
type candidateIntegrationCase struct {
	name string
	run  func(t *testing.T, graph accessHTTP)
}

// TestCandidateAccessHTTPProvesAssignmentIsolationAndRecovery covers T14 assignment A/B, retry and revocation.
func TestCandidateAccessHTTPProvesAssignmentIsolationAndRecovery(t *testing.T) {
	tests := []candidateIntegrationCase{
		{
			name: "one candidate two assignments keep isolated sessions",
			run: func(t *testing.T, graph accessHTTP) {
				hrCookie, csrf := loginHR(t, graph)
				candidateID := createCandidate(t, graph, hrCookie, csrf)
				assignmentA := insertAssignment(t, graph, candidateID, false, false)
				assignmentB := insertAssignment(t, graph, candidateID, false, false)
				tokenA := issueInvitation(t, graph, hrCookie, csrf, assignmentA)
				tokenB := issueInvitation(t, graph, hrCookie, csrf, assignmentB)
				cookieA := exchangeInvitation(t, graph, tokenA)
				cookieB := exchangeInvitation(t, graph, tokenB)
				sessionA := readCandidateSession(t, graph, cookieA)
				sessionB := readCandidateSession(t, graph, cookieB)
				require.Equal(t, assignmentA, sessionA.AssignmentID)
				require.Equal(t, assignmentB, sessionB.AssignmentID)
				require.NotEqual(t, sessionA.SessionID, sessionB.SessionID)
				require.Equal(t, candidateID, sessionA.CandidateID)
				require.Equal(t, candidateID, sessionB.CandidateID)
			},
		},
		{
			name: "lost exchange retry may create another session",
			run: func(t *testing.T, graph accessHTTP) {
				hrCookie, csrf := loginHR(t, graph)
				candidateID := createCandidate(t, graph, hrCookie, csrf)
				assignmentID := insertAssignment(t, graph, candidateID, false, false)
				token := issueInvitation(t, graph, hrCookie, csrf, assignmentID)
				first := exchangeInvitation(t, graph, token)
				second := exchangeInvitation(t, graph, token)
				require.NotEqual(t, first.Value, second.Value)
				require.Equal(t, assignmentID, readCandidateSession(t, graph, first).AssignmentID)
				require.Equal(t, assignmentID, readCandidateSession(t, graph, second).AssignmentID)
			},
		},
		{
			name: "reissue replaces invitation and keeps existing session",
			run: func(t *testing.T, graph accessHTTP) {
				hrCookie, csrf := loginHR(t, graph)
				candidateID := createCandidate(t, graph, hrCookie, csrf)
				assignmentID := insertAssignment(t, graph, candidateID, false, false)
				oldToken := issueInvitation(t, graph, hrCookie, csrf, assignmentID)
				cookie := exchangeInvitation(t, graph, oldToken)
				newToken := issueInvitation(t, graph, hrCookie, csrf, assignmentID)
				require.NotEqual(t, oldToken, newToken)
				require.Equal(t, http.StatusOK, doJSON(t, graph.handler, http.MethodGet, "/candidate/session", "", "", []*http.Cookie{cookie}).Code)
				unavailable := doJSON(t, graph.handler, http.MethodPost, "/candidate/access/exchange", `{"token":"`+oldToken+`"}`, "", nil)
				require.Equal(t, http.StatusGone, unavailable.Code)
				require.Equal(t, errs.IAInvitationUnavailable, decodeAccess(t, unavailable).Error.Code)
			},
		},
		{
			name: "assignment revocation denies exchange and session",
			run: func(t *testing.T, graph accessHTTP) {
				hrCookie, csrf := loginHR(t, graph)
				candidateID := createCandidate(t, graph, hrCookie, csrf)
				assignmentID := insertAssignment(t, graph, candidateID, false, false)
				token := issueInvitation(t, graph, hrCookie, csrf, assignmentID)
				cookie := exchangeInvitation(t, graph, token)
				revokeAssignment(t, graph, assignmentID)
				deniedExchange := doJSON(t, graph.handler, http.MethodPost, "/candidate/access/exchange", `{"token":"`+token+`"}`, "", nil)
				require.Equal(t, http.StatusGone, deniedExchange.Code)
				deniedSession := doJSON(t, graph.handler, http.MethodGet, "/candidate/session", "", "", []*http.Cookie{cookie})
				require.Equal(t, http.StatusUnauthorized, deniedSession.Code)
			},
		},
		{
			name: "completed assignment still allows session and does not start execution",
			run: func(t *testing.T, graph accessHTTP) {
				hrCookie, csrf := loginHR(t, graph)
				candidateID := createCandidate(t, graph, hrCookie, csrf)
				assignmentID := insertAssignment(t, graph, candidateID, false, true)
				token := issueInvitation(t, graph, hrCookie, csrf, assignmentID)
				cookie := exchangeInvitation(t, graph, token)
				require.Equal(t, assignmentID, readCandidateSession(t, graph, cookie).AssignmentID)
				require.Zero(t, countExecutions(t, graph))
			},
		},
		{
			name: "exact expiry refuses candidate session",
			run: func(t *testing.T, graph accessHTTP) {
				hrCookie, csrf := loginHR(t, graph)
				candidateID := createCandidate(t, graph, hrCookie, csrf)
				assignmentID := insertAssignment(t, graph, candidateID, false, false)
				token := issueInvitation(t, graph, hrCookie, csrf, assignmentID)
				cookie := exchangeInvitation(t, graph, token)
				graph.clock.set(graph.clock.Now().Add(2 * time.Hour))
				rec := doJSON(t, graph.handler, http.MethodGet, "/candidate/session", "", "", []*http.Cookie{cookie})
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := newCandidateAccessHTTP(t)
			tt.run(t, graph)
		})
	}
}

// candidateSessionView is the decoded GET /candidate/session payload used by isolation assertions.
type candidateSessionView struct {
	CandidateID  string
	AssignmentID string
	SessionID    string
}

// loginHR bootstraps ADMIN, provisions HR and returns the HR cookie and CSRF material.
func loginHR(t *testing.T, graph accessHTTP) (*http.Cookie, string) {
	t.Helper()
	adminEmail := "admin-" + uuid.NewString() + "@example.com"
	hrEmail := "hr-" + uuid.NewString() + "@example.com"
	bootstrapAdmin(t, graph.db, graph.clock, adminEmail)
	adminCookie := loginAdmin(t, graph, adminEmail)
	csrf := currentCSRF(t, graph, adminCookie)
	provision := doJSON(t, graph.handler, http.MethodPost, "/admin/hr-accounts",
		`{"name":"Sinta Wijaya","email":"`+hrEmail+`","password":"`+accessPass+`"}`, csrf, []*http.Cookie{adminCookie})
	require.Equal(t, http.StatusCreated, provision.Code, provision.Body.String())
	hrCookie := loginAdmin(t, graph, hrEmail)
	return hrCookie, currentCSRF(t, graph, hrCookie)
}

// createCandidate posts HR candidate identity and returns the new id.
func createCandidate(t *testing.T, graph accessHTTP, cookie *http.Cookie, csrf string) string {
	t.Helper()
	rec := doJSON(t, graph.handler, http.MethodPost, "/admin/candidates",
		`{"name":"Andi Pratama","email":"andi-`+uuid.NewString()+`@example.com"}`, csrf, []*http.Cookie{cookie})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var data struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(decodeAccess(t, rec).Data, &data))
	require.NotEmpty(t, data.ID)
	return data.ID
}

// issueInvitation issues a replacement invitation and returns the plaintext token.
func issueInvitation(t *testing.T, graph accessHTTP, cookie *http.Cookie, csrf, assignmentID string) string {
	t.Helper()
	rec := doJSON(t, graph.handler, http.MethodPost, "/admin/interview-assignments/"+assignmentID+"/invitations", "{}", csrf, []*http.Cookie{cookie})
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var data struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(decodeAccess(t, rec).Data, &data))
	require.NotEmpty(t, data.Token)
	return data.Token
}

// exchangeInvitation exchanges a live invitation and returns the candidate cookie.
func exchangeInvitation(t *testing.T, graph accessHTTP, token string) *http.Cookie {
	t.Helper()
	rec := doJSON(t, graph.handler, http.MethodPost, "/candidate/access/exchange", `{"token":"`+token+`"}`, "", nil)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	cookie := cookieMap(rec)["ia-candidate"]
	require.NotNil(t, cookie)
	return cookie
}

// readCandidateSession loads the current candidate session view.
func readCandidateSession(t *testing.T, graph accessHTTP, cookie *http.Cookie) candidateSessionView {
	t.Helper()
	rec := doJSON(t, graph.handler, http.MethodGet, "/candidate/session", "", "", []*http.Cookie{cookie})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var data struct {
		Candidate struct {
			ID string `json:"id"`
		} `json:"candidate"`
		Assignment struct {
			ID string `json:"id"`
		} `json:"assignment"`
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
	}
	require.NoError(t, json.Unmarshal(decodeAccess(t, rec).Data, &data))
	return candidateSessionView{CandidateID: data.Candidate.ID, AssignmentID: data.Assignment.ID, SessionID: data.Session.ID}
}
