// Package game is a game of Ships 3D as one player's console sees it: our ship, the other players
// eased toward where the server last saw them, and the bullets. It knows nothing of the screen or
// the network: a program gives it the server's messages and the controls, and sends what it
// returns. It plays as the browser does (ships-vue-3d, src/game/Game.ts): we test our own
// bullets against the ships as we see them, and the server checks the claim.
package game

import (
	"math"
	"strconv"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/protocol"
)

// How remote ships are drawn between snapshots: guessed ahead along their velocity for at most
// maximumGuessSeconds, and eased toward that guess at remoteEasing per second.
const (
	maximumGuessSeconds = 0.5
	remoteEasing        = 12
)

// muzzleShare is how far ahead of the ship a bullet starts, in ship radii.
const muzzleShare = 1.2

const millisecondsPerSecond = 1000

// stateSlackSeconds lets a state go a little early, so a beat that falls on a frame isn't missed
// by rounding.
const stateSlackSeconds = 0.001

// Remote is another player.
type Remote struct {
	Player   protocol.Player
	Shown    flight.State // where it is drawn
	reported protocol.PlayerState
	seenAt   float64 // seconds, when reported arrived
}

// Bullet is a shot in flight.
type Bullet struct {
	Id        string
	Position  flight.Vector
	Direction flight.Vector
	expiresAt float64
}

// Game is one game, from joining to leaving.
type Game struct {
	Settings protocol.Settings
	Id       string // ours, once welcomed
	Own      flight.State
	Player   protocol.Player // our name, ship, life and score
	Remotes  map[string]*Remote
	Bullets  []Bullet // ours
	Others   []Bullet // everyone else's
	Welcomed bool
	Alive    bool
	Problem  string // the server's last error message
	// SnapshotAt is when the server's last snapshot arrived, in seconds; HitAt when we were last
	// hit.
	SnapshotAt, HitAt float64

	diedAt    float64
	lastState float64
	lastFire  float64
	bulletIds int
	outgoing  []string
}

// New is a game waiting for the server's welcome.
func New() *Game {
	return &Game{Remotes: map[string]*Remote{}, lastFire: -1e9, lastState: -1e9}
}

// CanRespawn tells whether we are dead long enough to fly again.
func (game *Game) CanRespawn(now float64) bool {
	return game.Welcomed && !game.Alive && now-game.diedAt >= float64(game.Settings.RespawnDelayMs)/millisecondsPerSecond
}

// RespawnIn is how many seconds are left, at now, until we may fly again; 0 when we may.
func (game *Game) RespawnIn(now float64) float64 {
	return max(0, game.diedAt+float64(game.Settings.RespawnDelayMs)/millisecondsPerSecond-now)
}

// Handle applies one message from the server, at now seconds.
func (game *Game) Handle(message protocol.Incoming, states []protocol.PlayerState, now float64) {
	switch message.Type {
	case protocol.TypeWelcome:
		game.welcome(message, now)
	case protocol.TypePlayerJoined:
		game.addRemote(message.Player, now)
	case protocol.TypePlayerLeft:
		delete(game.Remotes, message.Id)
	case protocol.TypeSnapshot:
		game.SnapshotAt = now
		for _, state := range states {
			if remote, found := game.Remotes[state.Id]; found {
				remote.reported, remote.seenAt = state, now
			}
		}
	case protocol.TypeFire:
		game.Others = append(game.Others, Bullet{Id: message.Id, Position: message.Position, Direction: message.Direction, expiresAt: now + game.bulletSeconds()})
	case protocol.TypeDamage:
		game.setLife(message.Target, message.Life)
		if message.Target == game.Id {
			game.HitAt = now
		}
	case protocol.TypeDied:
		game.died(message.Target, message.From, now)
	case protocol.TypeRespawned:
		game.respawned(message, now)
	case protocol.TypeError:
		game.Problem = message.Message
	}
}

// welcome takes the rules, our id and everyone already playing.
func (game *Game) welcome(message protocol.Incoming, now float64) {
	game.Settings, game.Id, game.Welcomed = message.Settings, message.Id, true
	for _, player := range message.Players {
		if player.Id != game.Id {
			game.addRemote(player, now)
			continue
		}
		game.Player = player
		game.Own = flight.State{Position: player.Position, Rotation: player.Rotation}
		game.Alive = !player.Dead
	}
}

// addRemote starts showing another player where the server says they are.
func (game *Game) addRemote(player protocol.Player, now float64) {
	state := protocol.PlayerState{Id: player.Id, Position: player.Position, Rotation: player.Rotation, Velocity: player.Velocity}
	game.Remotes[player.Id] = &Remote{Player: player, Shown: flight.State{Position: player.Position, Rotation: player.Rotation}, reported: state, seenAt: now}
}

// setLife changes a player's life after damage.
func (game *Game) setLife(target string, life float64) {
	if target == game.Id {
		game.Player.Life = life
	} else if remote, found := game.Remotes[target]; found {
		remote.Player.Life = life
	}
}

// died marks target as destroyed and gives from the kill.
func (game *Game) died(target, from string, now float64) {
	if target == game.Id {
		game.Alive, game.diedAt = false, now
		game.Player.Deaths++
		game.Player.Life = 0
	} else if remote, found := game.Remotes[target]; found {
		remote.Player.Dead = true
		remote.Player.Deaths++
	}
	if from == game.Id {
		game.Player.Kills++
	} else if remote, found := game.Remotes[from]; found {
		remote.Player.Kills++
	}
}

// respawned brings a player back where the server placed them.
func (game *Game) respawned(message protocol.Incoming, now float64) {
	if message.Id == game.Id {
		game.Alive, game.Player.Life = true, message.Life
		game.Own = flight.State{Position: message.Position, Rotation: message.Rotation}
		return
	}
	if remote, found := game.Remotes[message.Id]; found {
		remote.Player.Dead, remote.Player.Life = false, message.Life
		remote.reported = protocol.PlayerState{Id: message.Id, Position: message.Position, Rotation: message.Rotation}
		remote.Shown = flight.State{Position: message.Position, Rotation: message.Rotation}
		remote.seenAt = now
	}
}

// bulletSeconds is how long a bullet flies.
func (game *Game) bulletSeconds() float64 {
	return float64(game.Settings.BulletTtlMs) / millisecondsPerSecond
}

// Step moves the game on by seconds to now, with the player's controls, and returns the messages
// to send. The slice is reused by the next call.
func (game *Game) Step(controls flight.Controls, firing, respawning bool, seconds, now float64) []string {
	game.outgoing = game.outgoing[:0]
	if !game.Welcomed {
		return game.outgoing
	}
	if game.Alive {
		flight.Step(&game.Own, controls, game.Settings.Settings, seconds)
		if firing && now-game.lastFire >= float64(game.Settings.FireCooldownMs)/millisecondsPerSecond {
			game.fire(now)
		}
		if interval := 1 / float64(game.Settings.TickRate); game.Settings.TickRate > 0 && now-game.lastState >= interval-stateSlackSeconds {
			// Keep the beat rather than restart it, so frames that don't divide it evenly
			// don't slow the states down; after a pause, start again from now.
			game.lastState = max(game.lastState+interval, now-interval)
			game.outgoing = append(game.outgoing, protocol.State(game.Own.Position, game.Own.Rotation, game.Own.Velocity()))
		}
	} else if respawning && game.CanRespawn(now) {
		game.diedAt = now // asked once; the server's answer brings us back
		game.outgoing = append(game.outgoing, protocol.Respawn())
	}
	game.moveRemotes(seconds, now)
	game.moveBullets(seconds, now)
	return game.outgoing
}

// fire shoots a bullet along our nose and reports it.
func (game *Game) fire(now float64) {
	game.lastFire = now
	game.bulletIds++
	direction := game.Own.Rotation.Apply(flight.Forward)
	bullet := Bullet{
		Id:        strconv.Itoa(game.bulletIds),
		Position:  game.Own.Position.Add(direction.Times(game.Settings.ShipRadius * muzzleShare)),
		Direction: direction,
		expiresAt: now + game.bulletSeconds(),
	}
	game.Bullets = append(game.Bullets, bullet)
	game.outgoing = append(game.outgoing, protocol.Fire(bullet.Id, bullet.Position, bullet.Direction))
}

// moveRemotes eases each remote ship toward where it probably is now.
func (game *Game) moveRemotes(seconds, now float64) {
	share := 1 - math.Exp(-remoteEasing*seconds)
	for _, remote := range game.Remotes {
		if remote.Player.Dead {
			continue
		}
		age := min(now-remote.seenAt, maximumGuessSeconds)
		guess := remote.reported.Position.Add(remote.reported.Velocity.Times(age))
		remote.Shown.Position = remote.Shown.Position.Toward(guess, share)
		remote.Shown.Rotation = remote.Shown.Rotation.Toward(remote.reported.Rotation, share)
	}
}

// moveBullets flies every bullet on, and tests ours against the ships as we draw them.
func (game *Game) moveBullets(seconds, now float64) {
	distance := game.Settings.BulletSpeed * seconds
	kept := game.Bullets[:0]
	for _, bullet := range game.Bullets {
		previous := bullet.Position
		bullet.Position = bullet.Position.Add(bullet.Direction.Times(distance))
		if target, hit := game.shipHit(previous, bullet.Position); hit {
			game.outgoing = append(game.outgoing, protocol.Hit(bullet.Id, target))
			continue
		}
		if now <= bullet.expiresAt {
			kept = append(kept, bullet)
		}
	}
	game.Bullets = kept
	others := game.Others[:0]
	for _, bullet := range game.Others {
		bullet.Position = bullet.Position.Add(bullet.Direction.Times(distance))
		if now <= bullet.expiresAt {
			others = append(others, bullet)
		}
	}
	game.Others = others
}

// shipHit finds the ship, as we draw it, that something moving from from to to passed through.
func (game *Game) shipHit(from, to flight.Vector) (target string, hit bool) {
	for id, remote := range game.Remotes {
		if !remote.Player.Dead && flight.SegmentHitsSphere(from, to, remote.Shown.Position, game.Settings.ShipRadius) {
			return id, true
		}
	}
	return "", false
}
