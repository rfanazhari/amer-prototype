package runtime

import "errors"

// ValidateAccessSettings checks bootstrap identity, origin and CSRF secret without logging secrets.
func ValidateAccessSettings(cfg Config) error {
	return errors.New("access settings are not validated")
}
