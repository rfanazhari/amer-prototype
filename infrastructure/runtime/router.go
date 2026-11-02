package runtime

import (
	"net/http"
	"strings"
)

// Router sends landing, interview and configuration paths to their handlers before access, administrative and candidate paths to access, and every other path to Task.
type Router struct {
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

// ServeHTTP sends each mounted path to its handler, /admin and /candidate catch-alls to access, and every other path to Task.
func (h Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Landing != nil && r.URL.Path == "/candidate/access" {
		h.Landing.ServeHTTP(w, r)
		return
	}
	if h.InterviewPage != nil && r.URL.Path == "/candidate/interview/page" {
		h.InterviewPage.ServeHTTP(w, r)
		return
	}
	if h.Interview != nil && (r.URL.Path == "/candidate/interview" || r.URL.Path == "/candidate/consents" || isCandidateExecutionPath(r.URL.Path)) {
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

// isCandidateExecutionPath matches only start, poll, takeover and cancel, leaving sibling routes unmounted.
func isCandidateExecutionPath(path string) bool {
	const base = "/candidate/executions"
	if path == base {
		return true
	}
	if !strings.HasPrefix(path, base+"/") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(path, base+"/"), "/")
	return parts[0] != "" && (len(parts) == 1 || (len(parts) == 2 && (parts[1] == "takeover" || parts[1] == "cancel")))
}
