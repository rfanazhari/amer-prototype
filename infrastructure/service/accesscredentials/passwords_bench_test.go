package accesscredentials_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
)

// BenchmarkPasswordsHash records Argon2id baseline cost on the local implementation host.
func BenchmarkPasswordsHash(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := (accesscredentials.Passwords{}).Hash(ctx, "correct-horse"); err != nil {
			b.Fatal(err)
		}
	}
}
