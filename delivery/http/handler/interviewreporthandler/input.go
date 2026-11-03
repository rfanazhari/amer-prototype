package interviewreporthandler

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

const maxBodyBytes = 1 << 20

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// prepare binds current administrative authority before origin, CSRF or request validation.
func (h Handler) prepare(w http.ResponseWriter, r *http.Request, post bool) (*http.Request, bool) {
	token, ok, candidate := accesshandler.ReadAdminCookie(r, h.SecureCookie)
	if !ok || candidate {
		fail(w, r, http.StatusUnauthorized, errs.IRPUnauthenticated)
		return r, false
	}
	r = accesshandler.BindAdministrativeCredential(r, token)
	session, account, err := h.Sessions.Resolve(r.Context())
	if err != nil {
		h.sendError(w, r, err)
		return r, false
	}
	if !service.AllowsInterviewReport(account.Role) || h.Origin == "" || r.Header.Get("Origin") != h.Origin {
		fail(w, r, http.StatusForbidden, errs.IRPForbidden)
		return r, false
	}
	if post {
		values := r.Header.Values("X-CSRF-Token")
		if len(values) != 1 || values[0] == "" || h.CSRF.Verify(service.CredentialAdministrative, session.ID, values[0]) != nil {
			fail(w, r, http.StatusForbidden, errs.IRPForbidden)
			return r, false
		}
	}
	return r, true
}

// parseInput validates canonical UUID syntax, body and headers after authority is established.
func parseInput(r *http.Request, rawID string, post bool) (string, error) {
	id, err := uuid.Parse(rawID)
	if err != nil || id == uuid.Nil || len(rawID) != 36 || !strings.EqualFold(rawID, id.String()) || r.URL.RawQuery != "" || r.URL.ForceQuery {
		return "", service.ErrReportInvalidInput
	}
	if !post {
		if r.Body != nil {
			body, err := io.ReadAll(io.LimitReader(r.Body, 1))
			if err != nil || len(body) != 0 {
				return "", service.ErrReportInvalidInput
			}
		}
		return id.String(), nil
	}
	keys := r.Header.Values("Idempotency-Key")
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if len(keys) != 1 || !keyPattern.MatchString(keys[0]) || err != nil || media != "application/json" || r.Body == nil {
		return "", service.ErrReportInvalidInput
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil || len(body) > maxBodyBytes {
		return "", service.ErrReportInvalidInput
	}
	// Only an empty object is accepted, so removing JSON whitespace must yield exactly {}.
	compact := bytes.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, body)
	if !bytes.Equal(compact, []byte("{}")) {
		return "", service.ErrReportInvalidInput
	}
	return id.String(), nil
}
