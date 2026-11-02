// Package voicesession names the failure the interview voice driver reports when its provider cannot complete setup.
package voicesession

import "errors"

// ErrSessionUnavailable is returned when the provider port cannot complete setup.
var ErrSessionUnavailable = errors.New("session unavailable")
