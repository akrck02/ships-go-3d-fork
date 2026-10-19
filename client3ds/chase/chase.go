// Package chase is the camera that follows our ship, as the browser's (ships-vue-3d,
// src/game/Game.ts): above and behind it, rolling with it, and a little late, so turns are felt
// instead of the world snapping around.
package chase

import (
	"math"

	"ships3d/client3ds/flight"
)

// Where the camera sits and looks, in ship radii, and how fast it catches up, per second.
const (
	heightShare     = 1.3
	backShare       = 3.6
	aheadShare      = 4
	positionCatchUp = 6
	rotationCatchUp = 8
)

// Camera is the chase camera's state between frames.
type Camera struct {
	Position flight.Vector
	Rotation flight.Rotation
	started  bool
}

// View is where the camera is and what it looks at.
type View struct {
	Eye, Target, Up flight.Vector
}

// Follow moves the camera on by seconds toward its place behind the ship, which is shipRadius
// big, and returns the view. The first call, and any after Reset, puts it there at once.
func (camera *Camera) Follow(ship flight.State, shipRadius, seconds float64) View {
	if !camera.started {
		camera.Position, camera.Rotation, camera.started = ship.Position, ship.Rotation, true
	}
	camera.Position = camera.Position.Toward(ship.Position, 1-math.Exp(-positionCatchUp*seconds))
	camera.Rotation = camera.Rotation.Toward(ship.Rotation, 1-math.Exp(-rotationCatchUp*seconds))
	rotation := camera.Rotation
	return View{
		Eye:    camera.Position.Add(rotation.Apply(flight.Vector{0, shipRadius * heightShare, -shipRadius * backShare})),
		Target: camera.Position.Add(rotation.Apply(flight.Vector{0, 0, shipRadius * aheadShare})),
		Up:     rotation.Apply(flight.Vector{0, 1, 0}),
	}
}

// Reset makes the next Follow jump to the ship: after a respawn, far from where it was.
func (camera *Camera) Reset() { camera.started = false }

// FieldOfView is the chase camera's vertical field of view, in radians: 58 degrees, between
// go-renderer's default of 45, which looked zoomed in, and the browser's 70, too wide for the
// 3DS's small screen.
const FieldOfView = 58 * math.Pi / 180

// screenAspect is the top screen's 400 × 240.
const screenAspect = 400.0 / 240.0

// Sees tells whether a sphere of radius at centre can show on the screen from this view, and is
// nearer than farthest: whether a ship there needs drawing at all. It errs on the side of yes,
// testing a cone around the screen's corners rather than its four edges.
func (view View) Sees(centre flight.Vector, radius, farthest float64) bool {
	toCentre := centre.Subtract(view.Eye)
	distance := toCentre.Length()
	if distance <= radius {
		return true // the camera is inside it
	}
	if distance-radius > farthest {
		return false
	}
	forward := view.Target.Subtract(view.Eye)
	forward = forward.Times(1 / forward.Length())
	verticalTangent := math.Tan(FieldOfView / 2)
	cornerAngle := math.Atan(math.Hypot(verticalTangent, verticalTangent*screenAspect))
	angle := math.Acos(max(-1, min(1, toCentre.Dot(forward)/distance)))
	return angle-math.Asin(min(1, radius/distance)) <= cornerAngle
}
