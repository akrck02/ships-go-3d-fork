//go:build nintendo3ds

package main

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/game"
	"ships3d/client3ds/impact"
)

// The force field's answer to a crash: the face of the world's edge that was struck lights up and
// fades, and a blue ring spreads from the point over the field's surface, as a shock wave over a
// force field. Both sit just inside the edge, so they don't flicker against its lines.
const (
	waveSegments    = 56
	waveWidthShare  = 0.02 // the ring's width for its distance from the camera
	waveLeastWidth  = 0.4
	fieldInsetShare = 0.998 // how far inside the edge the wave and the flash are drawn
	realWorldRadius = 1500  // the server's world (game/settings.go), which the wave's reach suits
	waveHeightShare = 0.08  // the band's height for the ring's radius, at its start
	waveTopShade    = 0.15  // the band's top edge, fading into space
)

var (
	waveColour  = [3]float32{0.2, 0.55, 1}
	flashColour = [3]float32{0.04, 0.16, 0.42}
)

// shockwave is the ring and the struck face.
type shockwave struct {
	ring, face *scene.Node
	struck     float64 // the crash they show (Game.EdgeHitAt)
	centre     flight.Vector
}

func newShockwave(space *scene.Scene) *shockwave {
	wave := &shockwave{
		ring: &scene.Node{Name: "shock wave", Hidden: true, Moving: true},
		face: &scene.Node{Name: "struck face", Hidden: true, Moving: true},
	}
	space.Nodes = append(space.Nodes, wave.face, wave.ring)
	return wave
}

// show spreads the ring and fades the face, for the last crash, seen from eye.
func (wave *shockwave) show(state *game.Game, triangles [][3]flight.Vector, eye flight.Vector, now float64) {
	if wave.ring.Mesh == nil {
		// Made when first needed, as the glow's mesh is.
		wave.ring.Mesh = &scene.Mesh{Name: "shock wave", Material: unlit()}
		wave.face.Mesh = &scene.Mesh{Name: "struck face", Material: unlit()}
	}
	if state.EdgeHitAt != 0 && state.EdgeHitAt != wave.struck {
		wave.struck, wave.centre = state.EdgeHitAt, state.EdgePoint
		lightFace(wave.face.Mesh, struckTriangle(triangles, state.EdgePoint))
	}
	age := now - wave.struck
	ringFade, faceFade := impact.Fade(age, impact.WaveSeconds), impact.Fade(age, impact.FlashSeconds)
	wave.ring.Hidden = wave.struck == 0 || ringFade == 0
	wave.face.Hidden = wave.struck == 0 || faceFade == 0 || len(triangles) == 0
	wave.face.Mesh.Material.BaseColor = graphics.Vec4{float32(faceFade), float32(faceFade), float32(faceFade), 1}
	wave.face.Transform = graphics.Identity()
	if wave.ring.Hidden {
		return
	}
	// The wave's reach is for the real world's size; a smaller world gets a smaller wave.
	reach := math.Min(1, wave.centre.Length()/realWorldRadius)
	waveRing(wave.ring.Mesh, wave.centre, impact.WaveRadius(age)*reach, eye, float32(ringFade))
	wave.ring.Transform = graphics.Identity()
}

// struckTriangle is the face of the edge whose middle is nearest the point.
func struckTriangle(triangles [][3]flight.Vector, point flight.Vector) [3]flight.Vector {
	nearest, best := [3]flight.Vector{}, math.Inf(1)
	for _, triangle := range triangles {
		middle := triangle[0].Add(triangle[1]).Add(triangle[2]).Times(1.0 / 3)
		if distance := middle.Subtract(point).Length(); distance < best {
			nearest, best = triangle, distance
		}
	}
	return nearest
}

// lightFace lays a face out just inside the edge, its middle brighter than its corners.
func lightFace(mesh *scene.Mesh, triangle [3]flight.Vector) {
	mesh.Positions, mesh.Normals, mesh.Colors = mesh.Positions[:0], mesh.Normals[:0], mesh.Colors[:0]
	middle := triangle[0].Add(triangle[1]).Add(triangle[2]).Times(1.0 / 3)
	for corner := range 3 {
		points := [3]flight.Vector{middle, triangle[corner], triangle[(corner+1)%3]}
		for index, point := range points {
			inside := point.Times(fieldInsetShare)
			brightness := float32(0.45)
			if index == 0 {
				brightness = 1
			}
			mesh.Positions = append(mesh.Positions, float32(inside[0]), float32(inside[1]), float32(inside[2]))
			mesh.Normals = append(mesh.Normals, 0, 1, 0)
			mesh.Colors = append(mesh.Colors, flashColour[0]*brightness, flashColour[1]*brightness, flashColour[2]*brightness, 1)
		}
	}
	mesh.Dirty = true
}

// waveRing lays out in mesh a circle of radius around centre on the edge's sphere, as wide as its
// distance from eye asks, its colour faded.
func waveRing(mesh *scene.Mesh, centre flight.Vector, radius float64, eye flight.Vector, fade float32) {
	mesh.Positions, mesh.Normals, mesh.Colors = mesh.Positions[:0], mesh.Normals[:0], mesh.Colors[:0]
	sphere := centre.Length()
	normal := centre.Times(1 / sphere)
	// Two directions across the surface at the centre.
	helper := flight.Vector{0, 1, 0}
	if math.Abs(normal[1]) > 0.9 {
		helper = flight.Vector{1, 0, 0}
	}
	across := cross(normal, helper)
	across = across.Times(1 / across.Length())
	along := cross(normal, across)
	width := max(waveLeastWidth, centre.Subtract(eye).Length()*waveWidthShare)
	onSphere := func(angle, distance float64) flight.Vector {
		flat := centre.Add(across.Times(math.Cos(angle) * distance)).Add(along.Times(math.Sin(angle) * distance))
		return flat.Times(sphere * fieldInsetShare / flat.Length())
	}
	colour := [3]float32{waveColour[0] * fade, waveColour[1] * fade, waveColour[2] * fade}
	// The wave stands up from the field, a band rising inward and falling as it spreads: the ship
	// usually scrapes along the edge, and a ring lying flat on it would be seen edge on.
	height := waveHeightShare * radius * float64(fade)
	quad := func(corners [4]flight.Vector, bottom, top float32) {
		shades := [4]float32{bottom, bottom, top, top}
		for _, index := range [6]int{0, 1, 2, 0, 2, 3} {
			corner := corners[index]
			mesh.Positions = append(mesh.Positions, float32(corner[0]), float32(corner[1]), float32(corner[2]))
			mesh.Normals = append(mesh.Normals, 0, 1, 0)
			mesh.Colors = append(mesh.Colors, colour[0]*shades[index], colour[1]*shades[index], colour[2]*shades[index], 1)
		}
	}
	raised := func(point flight.Vector) flight.Vector { return point.Subtract(normal.Times(height)) }
	for segment := range waveSegments {
		first := float64(segment) / waveSegments * 2 * math.Pi
		second := float64(segment+1) / waveSegments * 2 * math.Pi
		quad([4]flight.Vector{onSphere(first, radius-width/2), onSphere(second, radius-width/2), onSphere(second, radius+width/2), onSphere(first, radius+width/2)}, 1, 1)
		quad([4]flight.Vector{onSphere(first, radius), onSphere(second, radius), raised(onSphere(second, radius)), raised(onSphere(first, radius))}, 1, waveTopShade)
	}
	mesh.Dirty = true
}

func cross(first, second flight.Vector) flight.Vector {
	return flight.Vector{first[1]*second[2] - first[2]*second[1], first[2]*second[0] - first[0]*second[2], first[0]*second[1] - first[1]*second[0]}
}
