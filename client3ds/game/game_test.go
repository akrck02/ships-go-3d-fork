package game

import (
	"math"
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

func TestBeingHitIsRemembered(t *testing.T) {
	game := welcomed(50)
	game.Handle(protocol.Incoming{Type: protocol.TypeDamage, Target: "me", Life: 9}, nil, 3)
	if game.HitAt != 3 || game.Player.Life != 9 {
		t.Fatalf("hit at %v, life %v", game.HitAt, game.Player.Life)
	}
}

func TestTheRespawnCountsDown(t *testing.T) {
	game := welcomed(50)
	game.Handle(protocol.Incoming{Type: protocol.TypeDied, Target: "me", From: "them"}, nil, 10)
	if got := game.RespawnIn(11); got != 2 {
		t.Fatalf("%v seconds left, want 2", got)
	}
	if got := game.RespawnIn(20); got != 0 {
		t.Fatalf("%v seconds left after the delay, want 0", got)
	}
}

func TestRunningIntoTheEdgeIsNotedOnce(t *testing.T) {
	game := welcomed(5000)
	game.Own.Position = flight.Vector{0, 0, 1499}
	game.Own.Speed = 80
	game.Step(flight.Controls{}, false, false, 0.1, 1)
	if game.EdgeHitAt != 1 || game.EdgePoint[2] < 1499 {
		t.Fatalf("hit at %v, point %v", game.EdgeHitAt, game.EdgePoint)
	}
	game.Step(flight.Controls{}, false, false, 0.1, 1.1)
	if game.EdgeHitAt != 1 || game.EdgeTouchAt != 1.1 {
		t.Fatalf("scraping along the edge: hit %v, touch %v", game.EdgeHitAt, game.EdgeTouchAt)
	}
}

func TestHitsAndDeathsAreEventsWhereTheShipIs(t *testing.T) {
	game := welcomed(50)
	game.Remotes["them"].Player.Name = "Rival"
	game.Handle(protocol.Incoming{Type: protocol.TypeDamage, Target: "them", Life: 5}, nil, 1)
	game.Handle(protocol.Incoming{Type: protocol.TypeDied, Target: "me", From: "them"}, nil, 2)
	events := game.TakeEvents()
	if len(events) != 2 || events[0].Kind != ShipHit || events[0].Position[2] != 50 || events[0].Own {
		t.Fatalf("got %+v", events)
	}
	if events[1].Kind != ShipDestroyed || !events[1].Own || events[1].At != 2 {
		t.Fatalf("got %+v", events[1])
	}
	if game.KilledBy() != "Rival" {
		t.Fatalf("killed by %q", game.KilledBy())
	}
	if len(game.TakeEvents()) != 0 {
		t.Fatal("events came back twice")
	}
}

func TestTheEdgeIsFeltOnlyNearIt(t *testing.T) {
	game := welcomed(5000)
	for _, place := range []struct {
		distance, want float64
	}{{0, 0}, {1300, 0}, {1410, 0.5}, {1500, 1}} {
		game.Own.Position = flight.Vector{0, 0, place.distance}
		if got := game.EdgeNearness(); math.Abs(got-place.want) > 1e-9 {
			t.Errorf("at %v: nearness %v, want %v", place.distance, got, place.want)
		}
	}
}

func TestHeadingOutNearTheEdgeTurnsTheShipBackOnItsOwn(t *testing.T) {
	game := welcomed(5000)
	game.Own.Position = flight.Vector{0, 0, 1470}
	game.Own.Speed = 40
	pulledUp := flight.Controls{Pitch: 1}
	now := 1.0
	game.Step(pulledUp, false, false, 0.1, now)
	if !game.Returning {
		t.Fatal("heading out near the edge did not start the turn back")
	}
	for range 200 {
		now += 0.05
		game.Step(pulledUp, false, false, 0.05, now)
		if !game.Returning {
			break
		}
	}
	ahead := game.Own.Rotation.Apply(flight.Forward)
	toMiddle := game.Own.Position.Times(-1 / game.Own.Position.Length())
	if game.Returning || ahead.Dot(toMiddle) < returnFacing {
		t.Fatalf("still returning %v, facing the middle by %v", game.Returning, ahead.Dot(toMiddle))
	}
}

func TestHeadingInNearTheEdgeLeavesTheControlsAlone(t *testing.T) {
	game := welcomed(5000)
	game.Own.Position = flight.Vector{0, 0, 1450}
	game.Own.Rotation = flight.FromEulerXYZ(0, math.Pi, 0) // nose toward the middle
	game.Step(flight.Controls{}, false, false, 0.1, 1)
	if game.Returning {
		t.Fatal("a ship heading inward was turned")
	}
}
