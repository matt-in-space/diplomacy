package main

import (
	"context"
	"log"

	"github.com/matt-in-space/diplomacy/application/auth"
)

// seedDevUser creates a fixed development account so there's something to
// log in with immediately after starting the server with -seed. Errors are
// logged, not fatal — a dev convenience shouldn't crash the server, and
// "already exists" isn't reachable in practice today anyway since every
// repository is in-memory and starts empty each run.
func seedDevUser(ctx context.Context, authService *auth.Service) {
	const (
		email       = "user@example.com"
		displayName = "Dev User"
		password    = "password"
	)

	if _, err := authService.Signup(ctx, email, displayName, password); err != nil {
		log.Printf("seed: failed to create dev user %s: %v", email, err)
		return
	}
	log.Printf("seed: created dev user %s / %s", email, password)
}
