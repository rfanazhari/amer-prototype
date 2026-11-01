package runtime

import (
	"net/http"
	"strings"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/voicesessionhandler"
	"github.com/rfanazhari/amer-prototype/delivery/http/voicepage"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/realtimecall"
)

const (
	defaultRealtimeModel = "gpt-realtime-2.1"
	defaultRealtimeVoice = "marin"
	defaultRealtimeURL   = "https://api.openai.com/v1/realtime/calls"
)

// Router sends prototype voice paths to their handlers, configuration and candidate interview paths before access, administrative paths to access, and every other path to Task.
type Router struct {
	Page          http.Handler
	Setup         http.Handler
	Task          http.Handler
	Access        http.Handler
	Landing       http.Handler
	Configuration http.Handler
	// Interview serves the candidate interview JSON routes ahead of the access catch-all.
	Interview http.Handler
	// InterviewPage serves the candidate interview page shell ahead of the access catch-all.
	InterviewPage http.Handler
}

// isAgentConfigurationPath reports the configuration collections that must precede the access catch-all.
func isAgentConfigurationPath(path string) bool {
	return path == "/admin/agent-templates" || path == "/admin/agent-instances" || strings.HasPrefix(path, "/admin/agent-instances/")
}

// firstNonEmpty returns value when it is set and fallback otherwise.
func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

// assembleVoice builds the prototype page and setup handlers from optional provider settings.
func assembleVoice(cfg Config, logs *logmanager.Application) (http.Handler, http.Handler) {
	adapter := realtimecall.Adapter{
		APIKey: cfg.OpenAIAPIKey,
		Model:  firstNonEmpty(cfg.OpenAIRealtimeModel, defaultRealtimeModel),
		Voice:  firstNonEmpty(cfg.OpenAIRealtimeVoice, defaultRealtimeVoice),
		URL:    firstNonEmpty(cfg.OpenAIRealtimeURL, defaultRealtimeURL),
	}
	setup := voicesessionhandler.Handler{
		Negotiate: voicesession.Negotiate{Negotiator: adapter},
		LogError: func(err error) {
			txn := logs.Start("", "request error", logmanager.TxnTypeOther)
			txn.NoticeError(err)
			txn.End()
		},
	}
	return voicepage.Handler{}, setup
}

// ServeHTTP sends /voice-session and /voice-sessions to voice handlers, /admin to access, and every other path to Task.
func (h Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/voice-session":
		h.Page.ServeHTTP(w, r)
	case "/voice-sessions":
		h.Setup.ServeHTTP(w, r)
	default:
		if h.Landing != nil && r.URL.Path == "/candidate/access" {
			h.Landing.ServeHTTP(w, r)
			return
		}
		if h.InterviewPage != nil && r.URL.Path == "/candidate/interview/page" {
			h.InterviewPage.ServeHTTP(w, r)
			return
		}
		if h.Interview != nil && (r.URL.Path == "/candidate/interview" || r.URL.Path == "/candidate/consents") {
			h.Interview.ServeHTTP(w, r)
			return
		}
		if h.Configuration != nil && isAgentConfigurationPath(r.URL.Path) {
			h.Configuration.ServeHTTP(w, r)
			return
		}
		if h.Access != nil && (strings.HasPrefix(r.URL.Path, "/admin/") || strings.HasPrefix(r.URL.Path, "/candidate/")) {
			h.Access.ServeHTTP(w, r)
			return
		}
		h.Task.ServeHTTP(w, r)
	}
}
