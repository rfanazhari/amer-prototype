package runtime

import (
	"net/http"

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

// Router sends prototype voice paths to their handlers and every other path to Task.
type Router struct {
	Page  http.Handler
	Setup http.Handler
	Task  http.Handler
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

// ServeHTTP sends /voice-session and /voice-sessions to voice handlers and every other path to Task.
func (h Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/voice-session":
		h.Page.ServeHTTP(w, r)
	case "/voice-sessions":
		h.Setup.ServeHTTP(w, r)
	default:
		h.Task.ServeHTTP(w, r)
	}
}
