package accesscredentials

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/rfanazhari/amer-prototype/application/service"
	"golang.org/x/crypto/argon2"
)

const (
	// HashConcurrency is the process-wide Argon2id occupancy limit.
	// Local Apple M5 evidence for the OWASP 19 MiB / t=2 / p=1 baseline:
	// BenchmarkPasswordsHash ≈ 34ms and ≈19MiB per hash; occupancy stays at two.
	HashConcurrency = 2
	argonTime       = 2
	argonMemoryKiB  = 19 * 1024
	argonThreads    = 1
	argonKeyLen     = 32
	argonSaltLen    = 16
)

var (
	hashSlots          = make(chan struct{}, HashConcurrency)
	argon2idParamBlock = fmt.Sprintf("m=%d,t=%d,p=%d", argonMemoryKiB, argonTime, argonThreads)
)

var _ service.PasswordHasher = Passwords{}

// acquireHash occupies one process-wide Argon2id slot or returns cancellation.
func acquireHash(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	select {
	case hashSlots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseHash frees one process-wide Argon2id slot.
func releaseHash() {
	<-hashSlots
}

// parseEncoded accepts only the assessed Argon2id baseline encoding.
func parseEncoded(encoded string) (salt, key []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != fmt.Sprintf("v=%d", argon2.Version) || parts[3] != argon2idParamBlock {
		return nil, nil, service.ErrMalformedCredential
	}
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != argonSaltLen {
		return nil, nil, service.ErrMalformedCredential
	}
	key, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) != argonKeyLen {
		return nil, nil, service.ErrMalformedCredential
	}
	return salt, key, nil
}

// Passwords hashes and verifies administrative passwords using Argon2id.
type Passwords struct{}

// Hash returns an encoded Argon2id digest for password using the assessed baseline parameters.
func (Passwords) Hash(ctx context.Context, password string) (string, error) {
	if err := acquireHash(ctx); err != nil {
		return "", err
	}
	defer releaseHash()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$%s$%s$%s", argon2.Version, argon2idParamBlock, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// Verify compares encoded with password without leaking either value.
func (Passwords) Verify(ctx context.Context, encoded, password string) error {
	salt, key, err := parseEncoded(encoded)
	if err != nil {
		return err
	}
	if err := acquireHash(ctx); err != nil {
		return err
	}
	defer releaseHash()
	if err := ctx.Err(); err != nil {
		return err
	}
	computed := argon2.IDKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	if subtle.ConstantTimeCompare(key, computed) != 1 {
		return service.ErrAuthenticationFailed
	}
	return nil
}
