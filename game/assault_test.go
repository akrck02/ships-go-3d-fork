package game

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ships3d/models"
)

type arenaServer struct {
	*testServer
	arena *Arena
}

// newArenaServer runs an arena with the default rules changed by `change`.
// Teams are dealt in joining order (no shuffle): the first players to join
// are the gunners.
func newArenaServer(t *testing.T, change func(*AssaultRules)) *arenaServer {
	t.Helper()
	clock := &fakeClock{now: time.Unix(1_000_000, 0)}
	rules := DefaultAssaultRules()
	if change != nil {
		change(&rules)
	}
	loader := func(identity Identity, shipId string) (*models.Project3d, error) {
		if ship, ok := DefaultShip(shipId); ok {
			return ship, nil
		}
		return nil, ErrShipNotAllowed
	}
	identify := func(r *http.Request) Identity { return Identity{} }
	arena := NewArena(DefaultSettings(), rules, loader, identify, func(*http.Request) bool { return true })
	arena.now = clock.Now
	arena.shuffle = func(int, func(int, int)) {}
	server := httptest.NewServer(http.HandlerFunc(arena.ServeWS))
	t.Cleanup(server.Close)
	return &arenaServer{
		testServer: &testServer{clock: clock, url: "ws" + strings.TrimPrefix(server.URL, "http")},
		arena:      arena,
	}
}

// joinLobby connects a guest and waits until it is in the lobby.
func (s *arenaServer) joinLobby(t *testing.T, name string) (*testClient, map[string]any) {
	t.Helper()
	c := s.connect(t, "")
	c.send(map[string]any{"type": msgJoin, "shipId": "default:arrow", "name": name})
	lobby := c.expect(msgLobby)
	return c, lobby
}

// startRound fills a lobby with n players, lets its countdown run out and
// returns the clients, in joining order, with their roundStart messages.
func (s *arenaServer) startRound(t *testing.T, n int) ([]*testClient, []map[string]any) {
	t.Helper()
	clients := make([]*testClient, n)
	for i := range clients {
		clients[i], _ = s.joinLobby(t, fmt.Sprintf("p%d", i))
	}
	s.clock.Advance(time.Duration(s.arena.rules.LobbyWaitMs) * time.Millisecond)
	s.arena.tick()
	starts := make([]map[string]any, n)
	for i, c := range clients {
		starts[i] = c.expect(msgRoundStart)
	}
	return clients, starts
}

// teamOf finds a player's team in a roundStart message.
func teamOf(start map[string]any, id string) string {
	for _, p := range start["players"].([]any) {
		player := p.(map[string]any)
		if player["id"] == id {
			team, _ := player["team"].(string)
			return team
		}
	}
	return ""
}

func TestLobbyCountsDownFromTheLastJoin(t *testing.T) {
	s := newArenaServer(t, nil)
	clients := []*testClient{}
	for i := range 3 {
		c, lobby := s.joinLobby(t, fmt.Sprint(i))
		if _, counting := lobby["startsInMs"]; counting {
			t.Fatalf("counting down with %d players", i+1)
		}
		clients = append(clients, c)
	}
	fourth, lobby := s.joinLobby(t, "3")
	if lobby["startsInMs"] != float64(30000) || len(lobby["players"].([]any)) != 4 {
		t.Fatalf("lobby with 4 players: %v", lobby)
	}
	clients = append(clients, fourth)

	s.clock.Advance(20 * time.Second)
	s.arena.tick()
	// A fifth player restarts the wait.
	fifth, lobby := s.joinLobby(t, "4")
	if lobby["startsInMs"] != float64(30000) {
		t.Fatalf("countdown not restarted: %v", lobby["startsInMs"])
	}
	clients = append(clients, fifth)
	s.clock.Advance(20 * time.Second)
	s.arena.tick()
	fifth.expectNone(msgRoundStart)

	s.clock.Advance(10 * time.Second)
	s.arena.tick()
	gunners := 0
	for _, c := range clients {
		start := c.expect(msgRoundStart)
		if teamOf(start, start["id"].(string)) == TeamGunners {
			gunners++
		}
	}
	if gunners != 1 {
		t.Errorf("%d gunners for 5 players, want 1", gunners)
	}

	// The next player gets a lobby of their own.
	_, lobby = s.joinLobby(t, "late")
	if n := len(lobby["players"].([]any)); n != 1 {
		t.Errorf("new lobby has %d players", n)
	}
}

func TestLeavingTheLobbyStopsTheCountdown(t *testing.T) {
	s := newArenaServer(t, nil)
	var last *testClient
	for i := range 4 {
		last, _ = s.joinLobby(t, fmt.Sprint(i))
	}
	first, _ := s.joinLobby(t, "4")
	_ = first.conn.Close()
	lobby := last.expect(msgLobby)
	if _, counting := lobby["startsInMs"]; !counting {
		t.Fatal("4 players left, should still count down")
	}
}

func TestFullLobbyStartsAtOnce(t *testing.T) {
	s := newArenaServer(t, func(r *AssaultRules) { r.MaxPlayers = 5 })
	clients := []*testClient{}
	for i := range 5 {
		c := s.connect(t, "")
		c.send(map[string]any{"type": msgJoin, "shipId": "default:arrow", "name": fmt.Sprint(i)})
		clients = append(clients, c)
	}
	for _, c := range clients {
		c.expect(msgRoundStart)
	}
}

func TestAttackersDamageTheMothership(t *testing.T) {
	s := newArenaServer(t, func(r *AssaultRules) { r.MothershipLife = 2 })
	clients, starts := s.startRound(t, 4)
	gunner, attacker, other := clients[0], clients[1], clients[2]
	if teamOf(starts[0], starts[0]["id"].(string)) != TeamGunners {
		t.Fatal("first player should be the gunner")
	}
	mothership := starts[1]["mothership"].(map[string]any)
	if mothership["life"] != float64(2) || len(mothership["ship"].(map[string]any)["layers"].([]any)) == 0 {
		t.Fatalf("mothership: %v", mothership["life"])
	}

	attacker.place([3]float64{0, 0, -205})
	attacker.send(map[string]any{"type": msgFire, "id": "1", "p": []float64{0, 0, -200}, "d": []float64{0, 0, 1}})
	other.expect(msgFire)
	attacker.send(map[string]any{"type": msgHit, "bulletId": "1", "target": mothershipTarget})
	if damage := other.expect(msgDamage); damage["target"] != mothershipTarget || damage["life"] != float64(1) {
		t.Fatalf("damage: %v", damage)
	}

	// A shot that misses the hull by a mile doesn't count.
	s.clock.Advance(time.Second)
	attacker.send(map[string]any{"type": msgFire, "id": "2", "p": []float64{0, 0, -200}, "d": []float64{0, 1, 0}})
	other.expect(msgFire)
	attacker.send(map[string]any{"type": msgHit, "bulletId": "2", "target": mothershipTarget})
	other.expectNone(msgDamage)

	// Gunners can't hit their own ship.
	gunner.send(map[string]any{"type": msgFire, "id": "1", "p": starts[0]["players"].([]any)[0].(map[string]any)["p"], "d": []float64{0, 0, 1}})
	other.expect(msgFire)
	gunner.send(map[string]any{"type": msgHit, "bulletId": "1", "target": mothershipTarget})
	other.expectNone(msgDamage)

	s.clock.Advance(time.Second)
	attacker.send(map[string]any{"type": msgFire, "id": "3", "p": []float64{0, 0, -200}, "d": []float64{0, 0, 1}})
	other.expect(msgFire)
	attacker.send(map[string]any{"type": msgHit, "bulletId": "3", "target": mothershipTarget})
	if over := gunner.expect(msgRoundOver); over["winner"] != TeamAttackers {
		t.Fatalf("winner: %v", over["winner"])
	}

	// After the results, everybody is back in the lobby, with a newcomer.
	newcomer, _ := s.joinLobby(t, "new")
	s.clock.Advance(time.Duration(s.arena.rules.ResultsMs) * time.Millisecond)
	s.arena.tick()
	lobby := newcomer.expect(msgLobby)
	if n := len(lobby["players"].([]any)); n != 5 {
		t.Errorf("lobby has %d players after the round, want 5", n)
	}
	gunner.expect(msgLobby)
}

func TestGunnersShootDownAttackers(t *testing.T) {
	s := newArenaServer(t, nil)
	clients, starts := s.startRound(t, 4)
	gunner, attacker := clients[0], clients[1]
	attackerId := starts[1]["id"].(string)
	seat := starts[0]["players"].([]any)[0].(map[string]any)["seat"].(map[string]any)["p"].([]any)
	from := []float64{seat[0].(float64), seat[1].(float64), seat[2].(float64)}

	attacker.place([3]float64{from[0], from[1], from[2] + 100})
	for i := range 3 {
		s.clock.Advance(time.Second)
		gunner.send(map[string]any{"type": msgFire, "id": fmt.Sprint(i), "p": from, "d": []float64{0, 0, 1}})
		attacker.expect(msgFire)
		gunner.send(map[string]any{"type": msgHit, "bulletId": fmt.Sprint(i), "target": attackerId})
		attacker.expect(msgDamage)
	}
	if died := attacker.expect(msgDied); died["target"] != attackerId {
		t.Fatalf("died: %v", died)
	}

	attacker.send(map[string]any{"type": msgRespawn})
	attacker.expectNone(msgRespawned)
	s.clock.Advance(time.Duration(DefaultSettings().RespawnDelayMs) * time.Millisecond)
	attacker.send(map[string]any{"type": msgRespawn})
	if respawned := attacker.expect(msgRespawned); respawned["life"] != float64(3) {
		t.Errorf("respawned with %v life", respawned["life"])
	}
}

func TestCrashes(t *testing.T) {
	s := newArenaServer(t, nil)
	clients, starts := s.startRound(t, 5)
	gunner, a, b, c := clients[0], clients[1], clients[2], clients[3]
	gunnerId, bId := starts[0]["id"].(string), starts[2]["id"].(string)

	// Into the mothership: only when the server agrees we're at its hull.
	a.place([3]float64{0, 0, -400})
	a.send(map[string]any{"type": msgCrash, "with": mothershipTarget})
	c.expectNone(msgDied)
	a.place([3]float64{0, 0, -120})
	a.send(map[string]any{"type": msgCrash, "with": mothershipTarget})
	if died := c.expect(msgDied); died["target"] != starts[1]["id"] {
		t.Fatalf("died: %v", died)
	}

	gunner.expect(msgDied)

	// Gunners don't crash.
	b.send(map[string]any{"type": msgCrash, "with": gunnerId})
	gunner.expectNone(msgDied)

	// Two attackers.
	b.place([3]float64{500, 0, 0})
	c.place([3]float64{500, 0, 6})
	c.send(map[string]any{"type": msgCrash, "with": bId})
	gunner.expect(msgDied)
	gunner.expect(msgDied)
}

func TestMothershipHoldingOutWinsForTheGunners(t *testing.T) {
	s := newArenaServer(t, nil)
	clients, _ := s.startRound(t, 4)
	s.clock.Advance(time.Duration(s.arena.rules.RoundMs) * time.Millisecond)
	s.arena.tick()
	if over := clients[2].expect(msgRoundOver); over["winner"] != TeamGunners {
		t.Fatalf("winner: %v", over["winner"])
	}
}

func TestTeamWithNobodyLeftLoses(t *testing.T) {
	s := newArenaServer(t, nil)
	clients, _ := s.startRound(t, 4)
	_ = clients[0].conn.Close() // The only gunner.
	if over := clients[1].expect(msgRoundOver); over["winner"] != TeamAttackers {
		t.Fatalf("winner: %v", over["winner"])
	}
}

func TestGunnerCount(t *testing.T) {
	for players, want := range map[int]int{4: 1, 5: 1, 6: 2, 8: 2, 10: 3, 20: 5} {
		if got := gunnerCount(players, 3); got != want {
			t.Errorf("%d players: %d gunners, want %d", players, got, want)
		}
	}
}

func TestSeatsAreSpreadAndOnTheHull(t *testing.T) {
	a := newArenaServer(t, nil).arena
	for n := 1; n <= 5; n++ {
		taken := map[Seat]bool{}
		for _, seat := range seats(n, a.mothershipScale) {
			if taken[seat] {
				t.Fatalf("%d gunners share a seat", n)
			}
			taken[seat] = true
			// Right next to the hull: a short drop towards it reaches it.
			down := add(seat.P, vec3{0, -seat.Up * a.mothershipScale * 2, 0})
			if !a.hull.segmentHits(seat.P, down, 0) {
				t.Errorf("seat %v is not on the hull", seat)
			}
		}
	}
}

func TestMothershipDesignIsValid(t *testing.T) {
	if err := loadMothership().Validate(); err != nil {
		t.Fatal(err)
	}
}
