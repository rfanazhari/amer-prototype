package accesshandler

import (
	"net/http"
	"time"

	"github.com/rfanazhari/amer-prototype/application/service"
)

const (
	httpAdminCookie  = "ia-admin"
	httpsAdminCookie = "__Host-ia-admin"
	httpCandidate    = "ia-candidate"
	httpsCandidate   = "__Host-ia-candidate"
	sessionCookieAge = 2 * time.Hour
)

// adminCookieName selects the administrative cookie name for the configured transport.
func adminCookieName(secure bool) string {
	if secure {
		return httpsAdminCookie
	}
	return httpAdminCookie
}

// readAdminCookie returns the administrative credential or reports a candidate substitution.
func readAdminCookie(r *http.Request, secure bool) (string, bool, bool) {
	if cookie, err := r.Cookie(adminCookieName(secure)); err == nil && cookie.Value != "" {
		return cookie.Value, true, false
	}
	for _, name := range []string{httpCandidate, httpsCandidate} {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			return "", false, true
		}
	}
	return "", false, false
}

// bindCredential stores the trusted administrative cookie secret on the request context.
func bindCredential(r *http.Request, token string) *http.Request {
	return r.WithContext(service.WithAdministrativeCredential(r.Context(), token))
}

// ReadAdminCookie exposes the administrative cookie reader to sibling configuration routes.
func ReadAdminCookie(r *http.Request, secure bool) (string, bool, bool) {
	return readAdminCookie(r, secure)
}

// BindAdministrativeCredential exposes cookie-context binding to sibling configuration routes.
func BindAdministrativeCredential(r *http.Request, token string) *http.Request {
	return bindCredential(r, token)
}

// candidateCookieName selects the candidate cookie name for the configured transport.
func candidateCookieName(secure bool) string {
	if secure {
		return httpsCandidate
	}
	return httpCandidate
}

// readCandidateCookie returns the candidate credential or reports an administrative substitution.
func readCandidateCookie(r *http.Request, secure bool) (string, bool, bool) {
	if cookie, err := r.Cookie(candidateCookieName(secure)); err == nil && cookie.Value != "" {
		return cookie.Value, true, false
	}
	for _, name := range []string{httpAdminCookie, httpsAdminCookie} {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			return "", false, true
		}
	}
	return "", false, false
}

// bindCandidateCredential stores the trusted candidate cookie secret on the request context.
func bindCandidateCredential(r *http.Request, token string) *http.Request {
	return r.WithContext(service.WithCandidateCredential(r.Context(), token))
}

// setCandidateCookie writes the HttpOnly candidate session cookie after a committed exchange.
func setCandidateCookie(w http.ResponseWriter, secure bool, token string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 1 {
		maxAge = int(sessionCookieAge.Seconds())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     candidateCookieName(secure),
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// setSessionCookie writes the HttpOnly administrative session cookie after a committed login.
func setSessionCookie(w http.ResponseWriter, secure bool, token string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 1 {
		maxAge = int(sessionCookieAge.Seconds())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminCookieName(secure),
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie expires the administrative cookie without claiming server-side revocation.
func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminCookieName(secure),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0).UTC(),
	})
}
