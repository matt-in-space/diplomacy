package main

import (
	"context"
	"log"

	"github.com/matt-in-space/diplomacy/application/auth"
)

// devUser is a fixed development account seeded on startup with -seed.
type devUser struct {
	email       string
	displayName string
	password    string
}

// devUsers are the accounts seedDevUsers creates — two, so there's a
// second account on hand to test multi-player flows (joining a game) with,
// not just logging in as one user.
var devUsers = []devUser{
	{email: "user1@example.com", displayName: "Dev User", password: "password"},
	{email: "user2@example.com", displayName: "Dev User 2", password: "password"},
}

// seedDevUsers creates the fixed development accounts so there's something
// to log in with immediately after starting the server with -seed. Errors
// are logged, not fatal — a dev convenience shouldn't crash the server, and
// "already exists" isn't reachable in practice today anyway since every
// repository is in-memory and starts empty each run.
func seedDevUsers(ctx context.Context, authService *auth.Service) {
	for _, u := range devUsers {
		if _, err := authService.Signup(ctx, u.email, u.displayName, u.password); err != nil {
			log.Printf("seed: failed to create dev user %s: %v", u.email, err)
			continue
		}
		log.Printf("seed: created dev user %s / %s", u.email, u.password)
	}
}
