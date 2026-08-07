package main

import (
	"context"
	"testing"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
)

func TestSeedDevUsersCreatesLoginableAccounts(t *testing.T) {
	authService := auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
	ctx := context.Background()

	seedDevUsers(ctx, authService)

	for _, email := range []string{"user1@example.com", "user2@example.com"} {
		if _, err := authService.Login(ctx, email, "password"); err != nil {
			t.Fatalf("Login with seeded credentials for %s failed: %v", email, err)
		}
	}
}

func TestSeedDevUsersDoesNotPanicOnDuplicateCall(t *testing.T) {
	authService := auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
	ctx := context.Background()

	seedDevUsers(ctx, authService)
	seedDevUsers(ctx, authService) // duplicate emails — logs and returns, doesn't panic or crash
}
