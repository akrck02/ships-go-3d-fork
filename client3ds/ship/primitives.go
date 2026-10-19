// Package ship turns a ship's design into one mesh, made like the browser's (ships-vue-3d,
// src/shared/shipMesh.ts): the same twelve primitives with the same segments, turned and placed
// the same way, in flat colours.
package ship

import "math"

// point is a corner of a triangle, in float64 while the mesh is built.
type point [3]float64

// triangle is three corners.
type triangle [3]point

// The primitive types, as the server names them (models/project3d.go).
const (
	partBox      = "box"
	partSphere   = "sphere"
	partCylinder = "cylinder"
	partCone     = "cone"
	partWedge    = "wedge"
	partTorus    = "torus"
	partDome     = "dome"
	partCapsule  = "capsule"
	partPyramid  = "pyramid"
	partTube     = "tube"
	partPrism    = "prism"
	partFrustum  = "frustum"
)

// The torus's shape, and the sides of the shapes that have few.
const (
	torusRadius     = 0.35
	torusTubeRadius = 0.15
	pyramidSides    = 4
	prismSides      = 6
)

// Detail is how many segments the round primitives have.
type Detail struct {
	sphereWidth, sphereHeight int
	round                     int // cylinders, cones and frustums
	lathe                     int // domes, capsules and tubes
	torusTube, torusRing      int
	domeSteps, capsuleSteps   int
}

var (
	// Fine is the browser's detail, so a ship close by looks the same on both.
	Fine = Detail{sphereWidth: 20, sphereHeight: 14, round: 20, lathe: 20, torusTube: 10, torusRing: 28, domeSteps: 7, capsuleSteps: 5}
	// Coarse is for ships far away, where the screen shows a few pixels of them: about a fifth
	// of the triangles.
	Coarse = Detail{sphereWidth: 8, sphereHeight: 6, round: 8, lathe: 8, torusTube: 5, torusRing: 12, domeSteps: 3, capsuleSteps: 2}
	// Minimal is for designs with hundreds of round parts, which even coarse would make too many
	// triangles for the console's memory.
	Minimal = Detail{sphereWidth: 5, sphereHeight: 4, round: 5, lathe: 5, torusTube: 3, torusRing: 6, domeSteps: 1, capsuleSteps: 1}
)

// Levels are the details from the finest to the plainest.
var Levels = []Detail{Fine, Coarse, Minimal}

// primitiveKey is a type at a detail.
type primitiveKey struct {
	kind   string
	detail Detail
}

// primitives are each type's triangles, worked out once per detail.
var primitives = map[primitiveKey][]triangle{}

// primitive returns the triangles of a type, in the unit cube centred on the origin; nil for a
// type it doesn't know.
func primitive(kind string, detail Detail) []triangle {
	key := primitiveKey{kind, detail}
	if triangles, found := primitives[key]; found {
		return triangles
	}
	triangles := buildPrimitive(kind, detail)
	primitives[key] = triangles
	return triangles
}

func buildPrimitive(kind string, detail Detail) []triangle {
	switch kind {
	case partBox:
		return box()
	case partSphere:
		return sphere(0.5, detail.sphereWidth, detail.sphereHeight)
	case partCylinder:
		return cylinder(0.5, 0.5, detail.round, 0)
	case partCone:
		return cylinder(0, 0.5, detail.round, 0)
	case partWedge:
		return wedge()
	case partTorus:
		return torus(detail.torusRing, detail.torusTube)
	case partDome:
		return lathe(domeProfile(detail.domeSteps), detail.lathe)
	case partCapsule:
		return lathe(capsuleProfile(detail.capsuleSteps), detail.lathe)
	case partPyramid:
		return cylinder(0, math.Sqrt2/2, pyramidSides, math.Pi/4)
	case partTube:
		return lathe([]point2{{0.5, -0.5}, {0.5, 0.5}, {0.3, 0.5}, {0.3, -0.5}, {0.5, -0.5}}, detail.lathe)
	case partPrism:
		return cylinder(0.5, 0.5, prismSides, 0)
	case partFrustum:
		return cylinder(0.25, 0.5, detail.round, 0)
	}
	return nil
}

// quad is two triangles from four corners in order around the face.
func quad(first, second, third, fourth point) []triangle {
	return []triangle{{first, second, third}, {first, third, fourth}}
}

func box() []triangle {
	corner := func(x, y, z float64) point { return point{x * 0.5, y * 0.5, z * 0.5} }
	var triangles []triangle
	for axis := range 3 {
		for _, side := range []float64{-1, 1} {
			var corners [4]point
			for index, sign := range [4][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
				var coordinates [3]float64
				coordinates[axis] = side
				coordinates[(axis+1)%3] = sign[0]
				coordinates[(axis+2)%3] = sign[1]
				corners[index] = corner(coordinates[0], coordinates[1], coordinates[2])
			}
			triangles = append(triangles, quad(corners[0], corners[1], corners[2], corners[3])...)
		}
	}
	return triangles
}

// sphere is three.js's SphereGeometry: rings from the top (+Y) down.
func sphere(radius float64, widthSegments, heightSegments int) []triangle {
	at := func(column, row int) point {
		around := float64(column) / float64(widthSegments) * 2 * math.Pi
		down := float64(row) / float64(heightSegments) * math.Pi
		return point{-radius * math.Cos(around) * math.Sin(down), radius * math.Cos(down), radius * math.Sin(around) * math.Sin(down)}
	}
	var triangles []triangle
	for row := range heightSegments {
		for column := range widthSegments {
			first, second := at(column, row), at(column+1, row)
			third, fourth := at(column+1, row+1), at(column, row+1)
			if row != 0 {
				triangles = append(triangles, triangle{first, fourth, second})
			}
			if row != heightSegments-1 {
				triangles = append(triangles, triangle{second, fourth, third})
			}
		}
	}
	return triangles
}

// cylinder is three.js's CylinderGeometry of height 1, with both ends closed, its first side
// turned by startAngle about Y (a cone has a top radius of 0).
func cylinder(topRadius, bottomRadius float64, sides int, startAngle float64) []triangle {
	at := func(side int, radius, y float64) point {
		angle := float64(side)/float64(sides)*2*math.Pi + startAngle
		return point{radius * math.Sin(angle), y, radius * math.Cos(angle)}
	}
	var triangles []triangle
	for side := range sides {
		topFirst, topSecond := at(side, topRadius, 0.5), at(side+1, topRadius, 0.5)
		bottomFirst, bottomSecond := at(side, bottomRadius, -0.5), at(side+1, bottomRadius, -0.5)
		if topRadius > 0 {
			triangles = append(triangles, triangle{topFirst, bottomFirst, topSecond})
			triangles = append(triangles, triangle{point{0, 0.5, 0}, topFirst, topSecond})
		}
		if bottomRadius > 0 {
			triangles = append(triangles, triangle{bottomFirst, bottomSecond, topSecond})
			triangles = append(triangles, triangle{point{0, -0.5, 0}, bottomSecond, bottomFirst})
		}
	}
	return triangles
}

// wedge is a ramp: full height at the back (-Z), down to nothing at the front (+Z).
func wedge() []triangle {
	a, b := point{-0.5, -0.5, -0.5}, point{0.5, -0.5, -0.5}
	c, d := point{-0.5, 0.5, -0.5}, point{0.5, 0.5, -0.5}
	e, f := point{-0.5, -0.5, 0.5}, point{0.5, -0.5, 0.5}
	return []triangle{
		{a, b, f}, {a, f, e}, // bottom
		{a, c, d}, {a, d, b}, // back
		{c, e, f}, {c, f, d}, // slope
		{a, e, c}, // left side
		{b, d, f}, // right side
	}
}

// torus is three.js's TorusGeometry: a ring in the XY plane, around Z.
func torus(ringSegments, tubeSegments int) []triangle {
	at := func(ring, tube int) point {
		around := float64(ring) / float64(ringSegments) * 2 * math.Pi
		across := float64(tube) / float64(tubeSegments) * 2 * math.Pi
		distance := torusRadius + torusTubeRadius*math.Cos(across)
		return point{distance * math.Cos(around), distance * math.Sin(around), torusTubeRadius * math.Sin(across)}
	}
	var triangles []triangle
	for tube := range tubeSegments {
		for ring := range ringSegments {
			triangles = append(triangles, quad(at(ring, tube), at(ring+1, tube), at(ring+1, tube+1), at(ring, tube+1))...)
		}
	}
	return triangles
}

// point2 is a point of a lathe's profile: its distance from the Y axis and its height.
type point2 [2]float64

// lathe spins a profile around Y, as three.js's LatheGeometry.
func lathe(profile []point2, segments int) []triangle {
	at := func(segment int, profilePoint point2) point {
		angle := float64(segment) / float64(segments) * 2 * math.Pi
		return point{profilePoint[0] * math.Sin(angle), profilePoint[1], profilePoint[0] * math.Cos(angle)}
	}
	var triangles []triangle
	for segment := range segments {
		for index := 0; index+1 < len(profile); index++ {
			triangles = append(triangles, quad(at(segment, profile[index]), at(segment+1, profile[index]), at(segment+1, profile[index+1]), at(segment, profile[index+1]))...)
		}
	}
	return triangles
}

// domeProfile is half a sphere stretched to fill the cube, flat base at the bottom.
func domeProfile(steps int) []point2 {
	profile := []point2{{0, -0.5}}
	for step := 0; step <= steps; step++ {
		angle := float64(step) / float64(steps) * math.Pi / 2
		profile = append(profile, point2{0.5 * math.Cos(angle), -0.5 + math.Sin(angle)})
	}
	return profile
}

// capsuleProfile is a cylinder with rounded ends, each end a quarter of the height.
func capsuleProfile(steps int) []point2 {
	profile := []point2{{0, -0.5}}
	for step := 1; step <= steps; step++ {
		angle := float64(step) / float64(steps) * math.Pi / 2
		profile = append(profile, point2{0.5 * math.Sin(angle), -0.25 - 0.25*math.Cos(angle)})
	}
	for step := 0; step <= steps; step++ {
		angle := float64(step) / float64(steps) * math.Pi / 2
		profile = append(profile, point2{0.5 * math.Cos(angle), 0.25 + 0.25*math.Sin(angle)})
	}
	return profile
}
