package main

import (
	"context"
	"log"

	"github.com/matt-in-space/diplomacy/application/auth"
	"github.com/matt-in-space/diplomacy/application/lobby"
	"github.com/matt-in-space/diplomacy/core/gamemap"
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
// repository is in-memory and starts empty each run. Returns the created
// players, in devUsers order, so callers can seed further data (e.g. a dev
// game) against their real IDs — short of len(devUsers) if any signup
// failed.
func seedDevUsers(ctx context.Context, authService *auth.Service) []*auth.Player {
	players := make([]*auth.Player, 0, len(devUsers))
	for _, u := range devUsers {
		player, err := authService.Signup(ctx, u.email, u.displayName, u.password)
		if err != nil {
			log.Printf("seed: failed to create dev user %s: %v", u.email, err)
			continue
		}
		log.Printf("seed: created dev user %s / %s", u.email, u.password)
		players = append(players, player)
	}
	return players
}

// seedDevGame creates a pending game setup on the Western Europe map,
// hosted by the first dev user with the second already joined — so
// starting the server with -seed lands on a game one Start-Game click away
// from being playable, instead of having to repeat create-then-join by
// hand every restart. A no-op (logged, not fatal) if seedDevUsers didn't
// produce at least two players. Returns the created setup, or nil if
// seeding was skipped or failed — returned rather than discarded so
// callers (tests included) can verify what actually got created.
func seedDevGame(ctx context.Context, lobbyService *lobby.Service, players []*auth.Player) *lobby.GameSetup {
	if len(players) < 2 {
		log.Printf("seed: skipping dev game — need at least 2 dev users, got %d", len(players))
		return nil
	}
	host, joiner := players[0], players[1]

	setup, err := lobbyService.CreateGameSetup(ctx, host.ID, gamemap.WesternEuropeMapID)
	if err != nil {
		log.Printf("seed: failed to create dev game: %v", err)
		return nil
	}
	setup, err = lobbyService.JoinGameSetup(ctx, setup.InviteCode, joiner.ID)
	if err != nil {
		log.Printf("seed: failed to join dev game: %v", err)
		return nil
	}
	log.Printf("seed: created dev game /games/%s/lobby (host: %s, joined: %s)", setup.ID, host.Email, joiner.Email)
	return setup
}
