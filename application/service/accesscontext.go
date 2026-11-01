package service

import "context"

// administrativeCredentialKey isolates the trusted administrative cookie secret on context.
type administrativeCredentialKey struct{}

// candidateCredentialKey isolates the trusted candidate cookie secret on context.
type candidateCredentialKey struct{}

// WithAdministrativeCredential stores a trusted administrative session secret on ctx.
func WithAdministrativeCredential(ctx context.Context, plaintext string) context.Context {
	return context.WithValue(ctx, administrativeCredentialKey{}, plaintext)
}

// AdministrativeCredentialFrom returns the trusted administrative secret when present.
func AdministrativeCredentialFrom(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(administrativeCredentialKey{}).(string)
	return value, ok && value != ""
}

// WithCandidateCredential stores a trusted candidate session secret on ctx.
func WithCandidateCredential(ctx context.Context, plaintext string) context.Context {
	return context.WithValue(ctx, candidateCredentialKey{}, plaintext)
}

// CandidateCredentialFrom returns the trusted candidate secret when present.
func CandidateCredentialFrom(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(candidateCredentialKey{}).(string)
	return value, ok && value != ""
}
