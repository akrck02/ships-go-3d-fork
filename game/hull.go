package game

import (
	_ "embed"
	"encoding/json"
	"math"

	"ships3d/models"
)

// The mothership of the assault mode: a built-in design (the dorsal, wedge
// shaped capital ship from the project's ship designs), scaled up to
// AssaultRules.MothershipRadius and parked at the centre of the world with no
// rotation. It never moves, so the server can check hits against its actual
// shape rather than a sphere: every part is bounded by its scaled, rotated
// unit cube, and a bullet must pass through one of those boxes.
//
//go:embed mothership.json
var mothershipJSON []byte

func loadMothership() *models.Project3d {
	var project models.Project3d
	if err := json.Unmarshal(mothershipJSON, &project); err != nil {
		panic("game: mothership.json: " + err.Error())
	}
	return &project
}

// hullBox is one part's bounding box in world space: centre, half sizes
// along its own axes, and those axes (the rows of rot).
type hullBox struct {
	center vec3
	half   vec3
	rot    [3]vec3
}

// hull is the mothership as boxes, in world units.
type hull struct {
	boxes []hullBox
}

// newHull bounds every visible part of a design scaled by scale, mirrored
// parts twice, as the client draws them (shipMesh.ts).
func newHull(project *models.Project3d, scale float64) hull {
	h := hull{}
	for _, part := range project.VisibleParts() {
		h.boxes = append(h.boxes, partBox(part.Position, part.Rotation, part.Scale, scale))
		if part.Mirror {
			// mirroredTransform in shipModel.ts.
			p, r := part.Position, part.Rotation
			h.boxes = append(h.boxes, partBox(vec3{-p[0], p[1], p[2]}, vec3{r[0], -r[1], -r[2]}, part.Scale, scale))
		}
	}
	return h
}

func partBox(position, rotation, size vec3, scale float64) hullBox {
	return hullBox{
		center: mul(position, scale),
		half:   mul(size, scale/2),
		rot:    eulerXYZ(rotation),
	}
}

// eulerXYZ is the rotation matrix of an XYZ Euler angle, three.js' default
// order, as rows.
func eulerXYZ(r vec3) [3]vec3 {
	a, b := math.Cos(r[0]), math.Sin(r[0])
	c, d := math.Cos(r[1]), math.Sin(r[1])
	e, f := math.Cos(r[2]), math.Sin(r[2])
	ae, af, be, bf := a*e, a*f, b*e, b*f
	return [3]vec3{
		{c * e, -c * f, d},
		{af + be*d, ae - bf*d, -b * c},
		{bf - ae*d, be + af*d, a * c},
	}
}

// toLocal expresses a world point in the box's own frame.
func (b hullBox) toLocal(p vec3) vec3 {
	d := sub(p, b.center)
	// The inverse of a rotation is its transpose: dot with the columns.
	return vec3{
		d[0]*b.rot[0][0] + d[1]*b.rot[1][0] + d[2]*b.rot[2][0],
		d[0]*b.rot[0][1] + d[1]*b.rot[1][1] + d[2]*b.rot[2][1],
		d[0]*b.rot[0][2] + d[1]*b.rot[1][2] + d[2]*b.rot[2][2],
	}
}

// segmentHits reports whether the segment a-b passes within tolerance of the
// box (a slab test against the box grown by tolerance on every side).
func (b hullBox) segmentHits(a, end vec3, tolerance float64) bool {
	p, q := b.toLocal(a), b.toLocal(end)
	tMin, tMax := 0.0, 1.0
	for axis := 0; axis < 3; axis++ {
		half := b.half[axis] + tolerance
		d := q[axis] - p[axis]
		if math.Abs(d) < 1e-12 {
			if math.Abs(p[axis]) > half {
				return false
			}
			continue
		}
		t1, t2 := (-half-p[axis])/d, (half-p[axis])/d
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tMin, tMax = math.Max(tMin, t1), math.Min(tMax, t2)
		if tMin > tMax {
			return false
		}
	}
	return true
}

// distance is how far a point is from the box (0 inside it).
func (b hullBox) distance(p vec3) float64 {
	local := b.toLocal(p)
	outside := vec3{}
	for axis := 0; axis < 3; axis++ {
		outside[axis] = math.Max(0, math.Abs(local[axis])-b.half[axis])
	}
	return length(outside)
}

// sphereHits reports whether a sphere touches any part of the hull.
func (h hull) sphereHits(center vec3, radius float64) bool {
	for _, box := range h.boxes {
		if box.distance(center) <= radius {
			return true
		}
	}
	return false
}

func (h hull) segmentHits(a, b vec3, tolerance float64) bool {
	for _, box := range h.boxes {
		if box.segmentHits(a, b, tolerance) {
			return true
		}
	}
	return false
}

// Seat is where a gunner sits on the mothership, in world units, and which
// way is up for them: +1 on the dorsal hull, -1 hanging under it.
type Seat struct {
	P  vec3    `json:"p"`
	Up float64 `json:"up"`
}

// mothershipSeats are the gunners' turrets, in the design's units, in a ring
// around the ship that alternates between the top and the underside, so that
// taking every (n/k)th of them spreads k gunners evenly.
var mothershipSeats = []struct {
	x, z float64
	top  bool
}{
	{4, 12, true},
	{8, 0, false},
	{10.2, -12, true},
	{0, -19, false},
	{-10.2, -12, true},
	{-8, 0, false},
	{-4, 12, true},
	{0, 14, false},
}

// seats picks n seats spread around the ship, scaled to world units. The
// main hull is two flattened cones: 48 long, 30 wide, 4 high above the
// trench line and 3.5 below it at the stern, shrinking to nothing at the nose.
func seats(n int, scale float64) []Seat {
	if n <= 0 {
		return nil
	}
	result := make([]Seat, n)
	for i := range n {
		s := mothershipSeats[i*len(mothershipSeats)/n%len(mothershipSeats)]
		f := (24 - s.z) / 48
		width := 15 * f
		profile := 0.0
		if width > math.Abs(s.x) {
			profile = math.Sqrt(1 - (s.x/width)*(s.x/width))
		}
		y, up := -0.4-3.5*f*profile-0.6, -1.0
		if s.top {
			y, up = 0.4+4*f*profile+0.6, 1.0
		}
		result[i] = Seat{P: mul(vec3{s.x, y, s.z}, scale), Up: up}
	}
	return result
}
