package game

import (
	"strings"
	"testing"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/protocol"
)

var settings = protocol.Settings{
	Settings: flight.Settings{WorldRadius: 1500, MaxSpeed: 80, MinSpeed: -20, Acceleration: 40, PitchRate: 1.4, YawRate: 1, RollRate: 2.2},
	TickRate: 20, ShipRadius: 4, MaxLife: 10, BulletSpeed: 300, BulletTtlMs: 2000, FireCooldownMs: 150, RespawnDelayMs: 3000,
}

// welcomed is a game where we are "me" at the origin, facing +Z, and "them" waits ahead at z.
func welcomed(z float64) *Game {
	game := New()
	game.Handle(protocol.Incoming{Type: protocol.TypeWelcome, Id: "me", Settings: settings, Players: []protocol.Player{
		{Id: "me", Rotation: flight.NoRotation, Life: 10},
		{Id: "them", Position: flight.Vector{0, 0, z}, Rotation: flight.NoRotation, Life: 10},
	}}, nil, 0)
	return game
}

func sentOfType(messages []string, kind string) int {
	count := 0
	for _, message := range messages {
		if strings.Contains(message, `"type":"`+kind+`"`) {
			count++
		}
	}
	return count
}

func TestTheWelcomePlacesUsAndTheOthers(t *testing.T) {
	game := welcomed(50)
	if !game.Alive || game.Remotes["them"] == nil || game.Remotes["me"] != nil {
		t.Fatalf("alive %v, remotes %v", game.Alive, game.Remotes)
	}
}

func TestABulletAlongTheNoseHitsAShipAhead(t *testing.T) {
	game := welcomed(50)
	hits := 0
	for frame := range 30 {
		hits += sentOfType(game.Step(flight.Controls{}, frame == 0, false, 1.0/60, float64(frame)/60), protocol.TypeHit)
	}
	if hits != 1 {
		t.Fatalf("%d hits, want 1", hits)
	}
}

func TestFiringWaitsForTheCooldown(t *testing.T) {
	game := welcomed(5000)
	shots := 0
	for frame := range 60 { // one second, holding the trigger
		shots += sentOfType(game.Step(flight.Controls{}, true, false, 1.0/60, float64(frame)/60), protocol.TypeFire)
	}
	if shots != 7 { // at 0, 0.15, 0.30 … 0.90 seconds
		t.Fatalf("%d shots in a second, want 7", shots)
	}
}

func TestStatesGoAtTheTickRate(t *testing.T) {
	game := welcomed(5000)
	states := 0
	for frame := range 60 {
		states += sentOfType(game.Step(flight.Controls{}, false, false, 1.0/60, float64(frame)/60), protocol.TypeState)
	}
	if states < 19 || states > 21 {
		t.Fatalf("%d states in a second, want about 20", states)
	}
}

func TestADeadPlayerRespawnsOnlyAfterTheDelay(t *testing.T) {
	game := welcomed(50)
	game.Handle(protocol.Incoming{Type: protocol.TypeDied, Target: "me", From: "them"}, nil, 1)
	if game.Alive || game.Remotes["them"].Player.Kills != 1 {
		t.Fatal("dying didn't count")
	}
	if sentOfType(game.Step(flight.Controls{}, false, true, 0.1, 2), protocol.TypeRespawn) != 0 {
		t.Fatal("asked to respawn too soon")
	}
	if sentOfType(game.Step(flight.Controls{}, false, true, 0.1, 4.1), protocol.TypeRespawn) != 1 {
		t.Fatal("didn't ask to respawn after the delay")
	}
	game.Handle(protocol.Incoming{Type: protocol.TypeRespawned, Id: "me", Life: 10, Rotation: flight.NoRotation}, nil, 4.2)
	if !game.Alive || game.Player.Life != 10 {
		t.Fatal("not back after respawning")
	}
}

func TestRemoteShipsEaseTowardTheirSnapshot(t *testing.T) {
	game := welcomed(50)
	game.Handle(protocol.Incoming{Type: protocol.TypeSnapshot}, []protocol.PlayerState{{Id: "them", Position: flight.Vector{10, 0, 50}, Rotation: flight.NoRotation}}, 0)
	for frame := range 60 {
		game.Step(flight.Controls{}, false, false, 1.0/60, float64(frame)/60)
	}
	if x := game.Remotes["them"].Shown.Position[0]; x < 9.9 || x > 10 {
		t.Fatalf("x %v after a second, want almost 10", x)
	}
}
