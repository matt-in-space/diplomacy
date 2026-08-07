package main

import (
	"context"
	"testing"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/infrastructure/memory"
)

func TestSeedDevUserCreatesLoginableAccount(t *testing.T) {
	authService := auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
	ctx := context.Background()

	seedDevUser(ctx, authService)

	if _, err := authService.Login(ctx, "user@example.com", "password"); err != nil {
		t.Fatalf("Login with seeded credentials failed: %v", err)
	}
}

func TestSeedDevUserDoesNotPanicOnDuplicateCall(t *testing.T) {
	authService := auth.NewService(memory.NewPlayerRepository(), memory.NewSessionRepository())
	ctx := context.Background()

	seedDevUser(ctx, authService)
	seedDevUser(ctx, authService) // duplicate email — logs and returns, doesn't panic or crash
}
