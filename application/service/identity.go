package service

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// trimAdministrative removes leading and trailing Unicode Zs separators plus ASCII space and tab.
func trimAdministrative(value string) string {
	return strings.TrimFunc(value, func(r rune) bool {
		return r == ' ' || r == '\t' || unicode.Is(unicode.Zs, r)
	})
}

// hasControls reports C0, DEL or C1 characters that administrative fields must reject.
func hasControls(value string) bool {
	for _, r := range value {
		if r <= 0x1F || (r >= 0x7F && r <= 0x9F) {
			return true
		}
	}
	return false
}

// ldh reports whether r is an LDH domain-label character.
func ldh(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
}

// CanonicalAdministrativeName trims outer whitespace and accepts 1–120 Unicode code points without controls.
func CanonicalAdministrativeName(raw string) (string, error) {
	name := trimAdministrative(raw)
	n := utf8.RuneCountInString(name)
	if !utf8.ValidString(name) || n < 1 || n > 120 || hasControls(name) {
		return "", ErrInvalidInput
	}
	return name, nil
}

// CanonicalAdministrativeEmail trims, Unicode-lowercases and accepts a v1 LDH ASCII address.
func CanonicalAdministrativeEmail(raw string) (string, error) {
	email := strings.ToLower(trimAdministrative(raw))
	if !utf8.ValidString(email) || len(email) < 3 || len(email) > 254 {
		return "", ErrInvalidInput
	}
	for i := 0; i < len(email); i++ {
		if email[i] > 127 {
			return "", ErrInvalidInput
		}
	}
	if strings.Count(email, "@") != 1 || strings.Contains(email, "..") || strings.ContainsAny(email, " <>()[]\",;") {
		return "", ErrInvalidInput
	}
	local, domain, _ := strings.Cut(email, "@")
	if len(local) < 1 || len(local) > 64 || local == "" || domain == "" || strings.Contains(domain, "xn--") {
		return "", ErrInvalidInput
	}
	labels := strings.Split(domain, ".")
	if len(labels) == 0 {
		return "", ErrInvalidInput
	}
	for _, label := range labels {
		if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
			return "", ErrInvalidInput
		}
		for _, r := range label {
			if !ldh(r) {
				return "", ErrInvalidInput
			}
		}
	}
	return email, nil
}

// AcceptProvisionPassword accepts an 8–64 code-point secret without trimming or normalization.
func AcceptProvisionPassword(password string) error {
	n := utf8.RuneCountInString(password)
	if !utf8.ValidString(password) || n < 8 || n > 64 || hasControls(password) {
		return ErrInvalidInput
	}
	return nil
}

// AcceptLoginPassword treats the secret as opaque and rejects empty or overlong values.
func AcceptLoginPassword(password string) error {
	if password == "" {
		return ErrInvalidInput
	}
	n := utf8.RuneCountInString(password)
	if n < 1 || n > 64 {
		return ErrInvalidInput
	}
	return nil
}
