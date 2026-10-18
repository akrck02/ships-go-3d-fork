// Package radar places other ships on a map seen from above our own ship, as the browser's radar
// does (ships-vue-3d, src/game/radar.ts): the map turns with the ship, so ahead is always up.
package radar

import "ships3d/client3ds/flight"

// Ranges are the distances the map can show, in world units, as the browser's.
var Ranges = []float64{300, 600, 1200, 2400}

// DefaultRange is the index in Ranges a game starts with.
const DefaultRange = 1

// Blip is where a ship appears on the map, in shares of the map's radius: Right and Ahead on the
// map's plane, Above how far it is over (or, negative, under) that plane. A ship out of range is
// pinned to the edge, with InRange false.
type Blip struct {
	Right, Ahead, Above float64
	InRange             bool
}

// Place is the blip of something at target, seen from a ship at position facing rotation, on a map
// that shows distance world units from its centre to its edge.
func Place(target, position flight.Vector, rotation flight.Rotation, distance float64) Blip {
	// In the ship's frame +X is its left, +Y up and +Z ahead.
	local := rotation.Inverse().Apply(target.Subtract(position)).Times(1 / distance)
	blip := Blip{Right: -local[0], Ahead: local[2], Above: local[1], InRange: true}
	if length := local.Length(); length > 1 {
		blip = Blip{Right: blip.Right / length, Ahead: blip.Ahead / length, Above: blip.Above / length}
	}
	return blip
}
