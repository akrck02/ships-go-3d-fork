package game

import "ships3d/client3ds/flight"

// EventKind is what happened, for the screen to show it.
type EventKind int

const (
	// ShipHit is a bullet striking a ship: ours or another's.
	ShipHit EventKind = iota
	// ShipDestroyed is a ship blowing up.
	ShipDestroyed
)

// Event is something worth showing, where it happened and when (seconds); Own says it happened
// to our ship.
type Event struct {
	Kind     EventKind
	Position flight.Vector
	At       float64
	Own      bool
}

// mostEvents bounds the events waiting to be shown, should nobody take them.
const mostEvents = 64

// TakeEvents returns what happened since the last call, oldest first; the slice is reused.
func (game *Game) TakeEvents() []Event {
	taken := append(game.taken[:0], game.events...)
	game.events, game.taken = game.events[:0], taken
	return taken
}

// note keeps an event about the ship with that id, at where it is shown now.
func (game *Game) note(kind EventKind, id string, now float64) {
	if len(game.events) >= mostEvents {
		return
	}
	if id == game.Id {
		game.events = append(game.events, Event{Kind: kind, Position: game.Own.Position, At: now, Own: true})
		return
	}
	if remote, found := game.Remotes[id]; found {
		game.events = append(game.events, Event{Kind: kind, Position: remote.Shown.Position, At: now})
	}
}

// KilledBy is the name of whoever destroyed our ship last, "" when nobody (or the edge) did.
func (game *Game) KilledBy() string { return game.killedBy }
