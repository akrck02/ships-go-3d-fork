// Package protocol holds the game's websocket messages, as the server sends and reads them. It
// mirrors the server's game/protocol.go by hand, as the browser's src/game/protocol.ts does:
// change all three together.
package protocol

import (
	"encoding/json"
	"fmt"

	"ships3d/client3ds/flight"
)

// The message types, the "type" field of every message.
const (
	TypeJoin    = "join"
	TypeState   = "state"
	TypeFire    = "fire"
	TypeHit     = "hit"
	TypeRespawn = "respawn"

	TypeWelcome      = "welcome"
	TypePlayerJoined = "playerJoined"
	TypePlayerLeft   = "playerLeft"
	TypeSnapshot     = "snapshot"
	TypeDamage       = "damage"
	TypeDied         = "died"
	TypeRespawned    = "respawned"
	TypeError        = "error"
)

// Settings are the rules of the game, from the welcome.
type Settings struct {
	flight.Settings
	TickRate       int     `json:"tickRate"`
	ShipRadius     float64 `json:"shipRadius"`
	MaxLife        float64 `json:"maxLife"`
	BulletSpeed    float64 `json:"bulletSpeed"`
	BulletTtlMs    int     `json:"bulletTtlMs"`
	BulletDamage   float64 `json:"bulletDamage"`
	FireCooldownMs int     `json:"fireCooldownMs"`
	RespawnDelayMs int     `json:"respawnDelayMs"`
}

// Part is one primitive of a ship's design. Every primitive fills a unit cube centred on its
// position before its scale and rotation.
type Part struct {
	Type     string        `json:"type"`
	Position flight.Vector `json:"position"`
	Rotation flight.Vector `json:"rotation"` // Euler angles in radians, XYZ
	Scale    flight.Vector `json:"scale"`
	Color    string        `json:"color"`  // "#rrggbb"
	Mirror   bool          `json:"mirror"` // also drawn reflected across X = 0
}

// Layer groups parts in the editor; hidden layers aren't drawn.
type Layer struct {
	Name    string `json:"name"`
	Visible bool   `json:"visible"`
	Parts   []Part `json:"parts"`
}

// Ship is a design a player flies.
type Ship struct {
	Id     string  `json:"id"`
	Name   string  `json:"name"`
	Layers []Layer `json:"layers"`
}

// Player is everything shown about a player, sent when they join.
type Player struct {
	Id       string          `json:"id"`
	Name     string          `json:"name"`
	Guest    bool            `json:"guest"`
	Ship     Ship            `json:"ship"`
	Scale    float64         `json:"scale"`
	Life     float64         `json:"life"`
	Dead     bool            `json:"dead"`
	Kills    int             `json:"kills"`
	Deaths   int             `json:"deaths"`
	Position flight.Vector   `json:"p"`
	Rotation flight.Rotation `json:"q"`
	Velocity flight.Vector   `json:"v"`
}

// PlayerState is a moved player, in a snapshot.
type PlayerState struct {
	Id       string          `json:"id"`
	Position flight.Vector   `json:"p"`
	Rotation flight.Rotation `json:"q"`
	Velocity flight.Vector   `json:"v"`
}

// Incoming is any message from the server: Type says which fields are set.
type Incoming struct {
	Type     string        `json:"type"`
	Id       string        `json:"id"`
	Settings Settings      `json:"settings"`
	Players  []Player      `json:"players"` // welcome
	Player   Player        `json:"player"`  // playerJoined
	Owner    string        `json:"owner"`   // fire
	Position flight.Vector `json:"p"`
	// Rotation is a respawn's facing.
	Rotation  flight.Rotation `json:"q"`
	Direction flight.Vector   `json:"d"`
	Target    string          `json:"target"`
	From      string          `json:"from"`
	BulletId  string          `json:"bulletId"`
	Life      float64         `json:"life"`
	Message   string          `json:"message"`
}

// snapshot is read apart: its players are states, not the full players of a welcome.
type snapshot struct {
	Players []PlayerState `json:"players"`
}

// Read understands a message from the server. A snapshot's moved players come back in states.
func Read(text string) (message Incoming, states []PlayerState, err error) {
	if err = json.Unmarshal([]byte(text), &message); err != nil {
		return message, nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	if message.Type != TypeSnapshot {
		return message, nil, nil
	}
	// A snapshot's players don't fit Player's fields; they are read again as states.
	message.Players = nil
	var moved snapshot
	if err = json.Unmarshal([]byte(text), &moved); err != nil {
		return message, nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	return message, moved.Players, nil
}

// Join asks to play with a ship.
func Join(shipId, name string) string {
	return write(struct {
		Type   string `json:"type"`
		ShipId string `json:"shipId"`
		Name   string `json:"name"`
	}{TypeJoin, shipId, name})
}

// State reports where our ship is, ~20 times a second.
func State(position flight.Vector, rotation flight.Rotation, velocity flight.Vector) string {
	return write(struct {
		Type     string          `json:"type"`
		Position flight.Vector   `json:"p"`
		Rotation flight.Rotation `json:"q"`
		Velocity flight.Vector   `json:"v"`
	}{TypeState, position, rotation, velocity})
}

// Fire reports a bullet fired from position along the unit vector direction.
func Fire(id string, position, direction flight.Vector) string {
	return write(struct {
		Type      string        `json:"type"`
		Id        string        `json:"id"`
		Position  flight.Vector `json:"p"`
		Direction flight.Vector `json:"d"`
	}{TypeFire, id, position, direction})
}

// Hit reports that one of our bullets hit target.
func Hit(bulletId, target string) string {
	return write(struct {
		Type     string `json:"type"`
		BulletId string `json:"bulletId"`
		Target   string `json:"target"`
	}{TypeHit, bulletId, target})
}

// Respawn asks to fly again after dying.
func Respawn() string { return `{"type":"respawn"}` }

func write(message any) string {
	bytes, err := json.Marshal(message)
	if err != nil {
		// Only unencodable numbers (NaN, infinities) fail; the server would refuse them anyway.
		return `{"type":"invalid"}`
	}
	return string(bytes)
}
