package game_test

import "testing"

func TestGameCommitOrders(t *testing.T) {
	gm := loadWesternEuropeMap(t)
	g := newWesternEuropeGame(t, gm)

	if err := g.CommitOrders("eng", gm); err != nil {
		t.Fatalf("CommitOrders failed: %v", err)
	}
	if _, ok := g.CommittedOrders["eng"]; !ok {
		t.Fatal("CommittedOrders does not contain eng")
	}
}
