package runtime

import "net/http"

// Router sends prototype voice paths to their handlers and every other path to Task.
type Router struct {
	Page  http.Handler
	Setup http.Handler
	Task  http.Handler
}

// ServeHTTP currently forwards every path to Task so missing voice dispatch is observable.
func (h Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Task.ServeHTTP(w, r)
}
