package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/access"
	"github.com/rfanazhari/amer-prototype/domain/enum"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	accessOrigin = "http://localhost:8080"
	accessSecret = "abcdefghijklmnopqrstuvwxyz012345"
	accessPass   = "correct-horse"
)

// shiftClock is a test clock that can jump to an exact session expiry boundary.
type shiftClock struct {
	mu sync.Mutex
	at time.Time
}

// Now returns the frozen administrative clock instant.
func (c *shiftClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

// set replaces the frozen instant used by session lifetime checks.
func (c *shiftClock) set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = at.UTC().Truncate(time.Millisecond)
}

// accessHTTP is a real PostgreSQL-backed administrative HTTP graph.
type accessHTTP struct {
	db        *gorm.DB
	handler   accesshandler.Handler
	authority accessauthority.Authority
	clock     *shiftClock
}

// envelopeJSON decodes the public access envelope.
type envelopeJSON struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *struct {
		Code string `json:"code"`
	} `json:"error"`
}

// newAccessHTTP checks the versioned schema and wires real adapters for HTTP proof.
func newAccessHTTP(t *testing.T) accessHTTP {
	t.Helper()
	db := accessDB(t)
	clock := &shiftClock{at: time.Now().UTC().Truncate(time.Millisecond)}
	store := accessrepository.Store{DB: db}
	hasher := accesscredentials.Passwords{}
	tokens := accesscredentials.Tokens{}
	csrf := accesscredentials.CSRF{Secret: []byte(accessSecret)}
	sessions := accessauthority.Sessions{Store: store, Tokens: tokens, Clock: clock}
	return accessHTTP{
		db:        db,
		clock:     clock,
		authority: accessauthority.Authority{Sessions: sessions, Clock: clock},
		handler: accesshandler.Handler{
			Login:        accessusecase.Login{Accounts: store, Issuance: store, Hasher: hasher, Tokens: tokens, Clock: clock},
			Current:      accessquery.CurrentSession{Current: sessions, CSRF: csrf, Clock: clock},
			Provision:    accessusecase.ProvisionHR{Accounts: store, Current: sessions, Hasher: hasher, Clock: clock},
			Logout:       accessusecase.Logout{Current: sessions, Sessions: store, Clock: clock},
			CSRF:         csrf,
			Sessions:     sessions,
			Origin:       accessOrigin,
			SecureCookie: false,
			Limiter:      &accesshandler.LoginLimiter{},
		},
	}
}

// bootstrapAdmin runs bootstrap then inserts the ADMIN when the store is already initialized.
func bootstrapAdmin(t *testing.T, db *gorm.DB, clock service.Clock, email string) {
	t.Helper()
	ctx := context.Background()
	store := accessrepository.Store{DB: db}
	hasher := accesscredentials.Passwords{}
	require.NoError(t, (accessusecase.Bootstrap{
		Accounts: store,
		Hasher:   hasher,
		Clock:    clock,
		Name:     "Bootstrap Admin",
		Email:    email,
		Password: accessPass,
	}).Execute(ctx))
	_, err := store.FindByEmail(ctx, email)
	if err == nil {
		return
	}
	require.ErrorIs(t, err, accessport.ErrNotFound)
	hash, err := hasher.Hash(ctx, accessPass)
	require.NoError(t, err)
	now := clock.Now().UTC().Truncate(time.Millisecond)
	account, err := access.NewAdministrativeAccount(uuid.NewString(), enum.Admin, now)
	require.NoError(t, err)
	snapshot := account.Snapshot()
	require.NoError(t, store.WithinAccounts(ctx, func(tx accessport.AccountTx) error {
		return tx.Insert(ctx, accessport.AccountRecord{
			ID: snapshot.ID, Name: "Bootstrap Admin", Email: email, PasswordHash: hash,
			Role: snapshot.Role, Status: snapshot.Status, CreatedAt: snapshot.CreatedAt, UpdatedAt: snapshot.UpdatedAt,
		})
	}))
	_, err = store.FindByEmail(ctx, email)
	require.NoError(t, err)
}

// doJSON performs one administrative JSON request against the real handler.
func doJSON(t *testing.T, handler http.Handler, method, path, body, csrf string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.RemoteAddr = "192.0.2.80:9"
	req.Header.Set("Origin", accessOrigin)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// decodeAccess reads the envelope and requires no-store.
func decodeAccess(t *testing.T, rec *httptest.ResponseRecorder) envelopeJSON {
	t.Helper()
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	var view envelopeJSON
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view))
	return view
}

// cookieMap collects Set-Cookie values by name.
func cookieMap(rec *httptest.ResponseRecorder) map[string]*http.Cookie {
	out := map[string]*http.Cookie{}
	for _, cookie := range rec.Result().Cookies() {
		out[cookie.Name] = cookie
	}
	return out
}

// loginAdmin posts credentials and returns the session cookie.
func loginAdmin(t *testing.T, graph accessHTTP, email string) *http.Cookie {
	t.Helper()
	rec := doJSON(t, graph.handler, http.MethodPost, "/admin/sessions", `{"email":"`+email+`","password":"`+accessPass+`"}`, "", nil)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	cookie := cookieMap(rec)["ia-admin"]
	require.NotNil(t, cookie)
	require.NotEmpty(t, cookie.Value)
	return cookie
}

// currentCSRF reads csrfToken from GET /admin/session.
func currentCSRF(t *testing.T, graph accessHTTP, cookie *http.Cookie) string {
	t.Helper()
	rec := doJSON(t, graph.handler, http.MethodGet, "/admin/session", "", "", []*http.Cookie{cookie})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var data struct {
		CSRFToken string `json:"csrfToken"`
		Account   struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"account"`
	}
	view := decodeAccess(t, rec)
	require.NoError(t, json.Unmarshal(view.Data, &data))
	require.NotEmpty(t, data.CSRFToken)
	return data.CSRFToken
}

// TestAdministrativeHTTPHappyPathProvesBootstrapProvisionAuthorityAndLogout is the T9 access evidence path.
func TestAdministrativeHTTPHappyPathProvesBootstrapProvisionAuthorityAndLogout(t *testing.T) {
	graph := newAccessHTTP(t)
	adminEmail := "admin-" + uuid.NewString() + "@example.com"
	hrEmail := "hr-" + uuid.NewString() + "@example.com"
	bootstrapAdmin(t, graph.db, graph.clock, adminEmail)
	adminCookie := loginAdmin(t, graph, adminEmail)
	csrf := currentCSRF(t, graph, adminCookie)
	provision := doJSON(t, graph.handler, http.MethodPost, "/admin/hr-accounts",
		`{"name":"Sinta Wijaya","email":"`+hrEmail+`","password":"`+accessPass+`"}`, csrf, []*http.Cookie{adminCookie})
	require.Equal(t, http.StatusCreated, provision.Code, provision.Body.String())
	hrCookie := loginAdmin(t, graph, hrEmail)
	hrCSRF := currentCSRF(t, graph, hrCookie)
	hrCtx := service.WithAdministrativeCredential(context.Background(), hrCookie.Value)
	principal, err := graph.authority.Principal(hrCtx)
	require.NoError(t, err)
	require.NoError(t, graph.authority.Authorize(hrCtx, principal, service.AgentConfigurationRead, ""))
	adminCtx := service.WithAdministrativeCredential(context.Background(), adminCookie.Value)
	adminPrincipal, err := graph.authority.Principal(adminCtx)
	require.NoError(t, err)
	require.ErrorIs(t, graph.authority.Authorize(adminCtx, adminPrincipal, service.AgentConfigurationRead, ""), service.ErrForbidden)
	logout := doJSON(t, graph.handler, http.MethodDelete, "/admin/sessions/current", "", hrCSRF, []*http.Cookie{hrCookie})
	require.Equal(t, http.StatusOK, logout.Code)
	denied := doJSON(t, graph.handler, http.MethodGet, "/admin/session", "", "", []*http.Cookie{hrCookie})
	require.Equal(t, http.StatusUnauthorized, denied.Code)
	require.Equal(t, errs.IAUnauthenticated, decodeAccess(t, denied).Error.Code)
}

// accessDenialCase names one real HTTP denial required by administrative evidence.
type accessDenialCase struct {
	name string
	run  func(t *testing.T, graph accessHTTP, db *gorm.DB)
}

// TestAdministrativeHTTPDenialsCoverExpiryRevocationAndCandidateCookies records T9 failure evidence.
func TestAdministrativeHTTPDenialsCoverExpiryRevocationAndCandidateCookies(t *testing.T) {
	tests := []accessDenialCase{
		{
			name: "invalid credentials issue no cookie",
			run: func(t *testing.T, graph accessHTTP, db *gorm.DB) {
				email := "admin-" + uuid.NewString() + "@example.com"
				bootstrapAdmin(t, db, graph.clock, email)
				rec := doJSON(t, graph.handler, http.MethodPost, "/admin/sessions", `{"email":"`+email+`","password":"wrong-horse"}`, "", nil)
				require.Equal(t, http.StatusUnauthorized, rec.Code)
				require.Empty(t, cookieMap(rec)["ia-admin"])
			},
		},
		{
			name: "duplicate hr email is conflict",
			run: func(t *testing.T, graph accessHTTP, db *gorm.DB) {
				adminEmail := "admin-" + uuid.NewString() + "@example.com"
				hrEmail := "hr-" + uuid.NewString() + "@example.com"
				bootstrapAdmin(t, db, graph.clock, adminEmail)
				cookie := loginAdmin(t, graph, adminEmail)
				csrf := currentCSRF(t, graph, cookie)
				body := `{"name":"Sinta Wijaya","email":"` + hrEmail + `","password":"` + accessPass + `"}`
				first := doJSON(t, graph.handler, http.MethodPost, "/admin/hr-accounts", body, csrf, []*http.Cookie{cookie})
				require.Equal(t, http.StatusCreated, first.Code, first.Body.String())
				second := doJSON(t, graph.handler, http.MethodPost, "/admin/hr-accounts", body, csrf, []*http.Cookie{cookie})
				require.Equal(t, http.StatusConflict, second.Code)
				require.Equal(t, errs.IAAccountConflict, decodeAccess(t, second).Error.Code)
			},
		},
		{
			name: "candidate cookie is excluded",
			run: func(t *testing.T, graph accessHTTP, _ *gorm.DB) {
				rec := doJSON(t, graph.handler, http.MethodGet, "/admin/session", "", "", []*http.Cookie{{Name: "ia-candidate", Value: "not-admin"}})
				require.Equal(t, http.StatusUnauthorized, rec.Code)
				require.Equal(t, errs.IAUnauthenticated, decodeAccess(t, rec).Error.Code)
			},
		},
		{
			name: "exact expiry refuses current session",
			run: func(t *testing.T, graph accessHTTP, db *gorm.DB) {
				email := "admin-" + uuid.NewString() + "@example.com"
				bootstrapAdmin(t, db, graph.clock, email)
				cookie := loginAdmin(t, graph, email)
				graph.clock.set(graph.clock.Now().Add(2 * time.Hour))
				rec := doJSON(t, graph.handler, http.MethodGet, "/admin/session", "", "", []*http.Cookie{cookie})
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
		{
			name: "restart keeps committed session until logout",
			run: func(t *testing.T, graph accessHTTP, db *gorm.DB) {
				email := "admin-" + uuid.NewString() + "@example.com"
				bootstrapAdmin(t, db, graph.clock, email)
				cookie := loginAdmin(t, graph, email)
				restarted := newAccessHTTP(t)
				restarted.clock.set(graph.clock.Now())
				rec := doJSON(t, restarted.handler, http.MethodGet, "/admin/session", "", "", []*http.Cookie{cookie})
				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := newAccessHTTP(t)
			tt.run(t, graph, graph.db)
		})
	}
}

// TestConcurrentHRProvisionCollapsesToOneAccount proves uniqueness under concurrent HTTP writes.
func TestConcurrentHRProvisionCollapsesToOneAccount(t *testing.T) {
	graph := newAccessHTTP(t)
	adminEmail := "admin-" + uuid.NewString() + "@example.com"
	hrEmail := "hr-" + uuid.NewString() + "@example.com"
	bootstrapAdmin(t, graph.db, graph.clock, adminEmail)
	cookie := loginAdmin(t, graph, adminEmail)
	csrf := currentCSRF(t, graph, cookie)
	body := `{"name":"Sinta Wijaya","email":"` + hrEmail + `","password":"` + accessPass + `"}`
	results := make(chan int, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			rec := doJSON(t, graph.handler, http.MethodPost, "/admin/hr-accounts", body, csrf, []*http.Cookie{cookie})
			results <- rec.Code
		}()
	}
	wg.Wait()
	close(results)
	var created, conflicted int
	for code := range results {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicted++
		}
	}
	require.Equal(t, 1, created)
	require.Equal(t, 1, conflicted)
}
