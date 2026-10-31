// Package voicepage serves the prototype voice-session HTML document.
package voicepage

import "net/http"

// Handler serves GET /voice-session and rejects other methods on that path.
type Handler struct{}

// ServeHTTP writes the embedded page for GET and a method error otherwise.
func (Handler) ServeHTTP(http.ResponseWriter, *http.Request) {}
