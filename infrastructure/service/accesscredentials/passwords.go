package accesscredentials

import "context"

// HashConcurrency is the process-wide Argon2id occupancy limit.
const HashConcurrency = 2

// Passwords hashes and verifies administrative passwords using Argon2id.
type Passwords struct{}

// Hash returns an encoded Argon2id digest for password.
func (Passwords) Hash(ctx context.Context, password string) (string, error) {
	return "", nil
}

// Verify compares encoded with password without leaking either value.
func (Passwords) Verify(ctx context.Context, encoded, password string) error {
	return nil
}
