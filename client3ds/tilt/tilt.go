// Package tilt steers with the console itself, as the browser steers with a phone's tilt
// (ships-vue-3d, src/game/tilt.ts): tipping the top screen toward the player raises the nose,
// turning the console like a steering wheel turns the ship. The 3DS has a gyroscope, which
// tells how fast it turns rather than where it points, so the angles are added up from the
// rates, from where the player last centred it.
package tilt

import "math"

// The browser's feel: nothing within the dead zone, full deflection at full.
const (
	deadZoneDegrees = 3
	fullDegrees     = 25
)

// Tilt is how far the console has turned since it was centred, in degrees.
type Tilt struct {
	Pitch float64 // the top screen toward the player is positive
	Steer float64 // clockwise, as the player sees it, is positive
}

// Turn adds a frame's turning: the rates in degrees a second, over seconds.
func (tilt *Tilt) Turn(pitchRate, steerRate, seconds float64) {
	tilt.Pitch += pitchRate * seconds
	tilt.Steer += steerRate * seconds
}

// Centre makes the console's present position the neutral one.
func (tilt *Tilt) Centre() { *tilt = Tilt{} }

// Axis maps an angle to a control in [-1, 1], with the browser's gentle curve, so small
// corrections stay small.
func Axis(degrees float64) float64 {
	magnitude := math.Abs(degrees)
	if magnitude <= deadZoneDegrees {
		return 0
	}
	share := min(1, (magnitude-deadZoneDegrees)/(fullDegrees-deadZoneDegrees))
	// The browser's blend of a straight line and a square, unchanged.
	const straight, squared = 0.4, 0.6
	return math.Copysign(straight*share+squared*share*share, degrees)
}
