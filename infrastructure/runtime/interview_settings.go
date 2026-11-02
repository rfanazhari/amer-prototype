package runtime

import (
	"errors"
	"net/url"
	"strings"
)

// ValidateInterviewSettings rejects malformed optional Live endpoint and model settings before startup I/O.
func ValidateInterviewSettings(cfg Config) error {
	for _, setting := range []struct {
		raw    string
		socket bool
	}{{cfg.OpenAILiveURL, false}, {cfg.OpenAILiveAttachURL, true}} {
		if setting.raw == "" {
			continue
		}
		parsed, err := url.Parse(setting.raw)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return errors.New("invalid Live provider endpoint")
		}
		if setting.socket && parsed.Scheme != "ws" && parsed.Scheme != "wss" {
			return errors.New("invalid Live sideband endpoint scheme")
		}
		if !setting.socket && parsed.Scheme != "http" && parsed.Scheme != "https" {
			return errors.New("invalid Live session endpoint scheme")
		}
	}
	for _, value := range []string{cfg.OpenAILiveModel, cfg.OpenAILiveVoice} {
		if value != "" && (strings.TrimSpace(value) != value || strings.ContainsAny(value, "\n\r\t ")) {
			return errors.New("invalid Live model or voice")
		}
	}
	return nil
}
