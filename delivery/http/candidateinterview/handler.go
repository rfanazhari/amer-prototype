package candidateinterview

import "net/http"

// Handler serves GET /candidate/interview/page.
type Handler struct{}

// ServeHTTP is not implemented yet.
func (Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
