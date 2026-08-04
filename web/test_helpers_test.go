package web_test

import (
	"testing"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
)

// newTestAuthService returns an auth.Service backed by fresh, empty
// in-memory repositories — isolated per test, same as every other test in
// this codebase that exercises real repository round-tripping rather than
// a call-counting fake.
func newTestAuthService(t *testing.T) *auth.Service {
	t.Helper()
	return auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
}
