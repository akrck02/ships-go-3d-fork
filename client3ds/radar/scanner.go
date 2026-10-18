package radar

import "math"

// ScanSeconds is how long the sweep takes to go round once.
const ScanSeconds = 5.0

// Echo is a ship as the sweep last saw it: where, and when (seconds).
type Echo struct {
	Blip   Blip
	SeenAt float64
}

// Scanner is a classic radar's sweep: a line turning clockwise from straight ahead, once every
// ScanSeconds. A ship shows where the line last passed over it, and fades until it passes again.
type Scanner struct {
	Echoes  map[string]Echo
	started bool
	angle   float64 // where the line was at the previous Update, in radians clockwise from ahead
}

// NewScanner is a sweep that has seen nothing yet.
func NewScanner() *Scanner { return &Scanner{Echoes: map[string]Echo{}} }

// Angle is where the line is at now seconds, in radians clockwise from ahead.
func Angle(now float64) float64 {
	return math.Mod(now, ScanSeconds) / ScanSeconds * 2 * math.Pi
}

// Update turns the line to now and records each ship it passed over since the last call. blips are
// where the ships are now, by player id; ships no longer there are forgotten.
func (scanner *Scanner) Update(now float64, blips map[string]Blip) {
	angle := Angle(now)
	previous := scanner.angle
	if !scanner.started {
		previous, scanner.started = angle, true
	}
	scanner.angle = angle
	for id := range scanner.Echoes {
		if _, found := blips[id]; !found {
			delete(scanner.Echoes, id)
		}
	}
	for id, blip := range blips {
		if passedOver(previous, angle, bearing(blip)) {
			scanner.Echoes[id] = Echo{Blip: blip, SeenAt: now}
		}
	}
}

// Brightness is how bright an echo still is at now: 1 when just seen, 0 a whole turn later.
func (echo Echo) Brightness(now float64) float64 {
	return max(0, 1-(now-echo.SeenAt)/ScanSeconds)
}

// bearing is the blip's direction, in radians clockwise from ahead, from 0 to a whole turn.
func bearing(blip Blip) float64 {
	angle := math.Atan2(blip.Right, blip.Ahead)
	if angle < 0 {
		angle += 2 * math.Pi
	}
	return angle
}

// passedOver tells whether a line turning clockwise from from to to went over angle, also when it
// went past a whole turn and started again.
func passedOver(from, to, angle float64) bool {
	if from <= to {
		return from < angle && angle <= to
	}
	return angle > from || angle <= to
}
