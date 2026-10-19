//go:build nintendo3ds

package main

import (
	"math/rand/v2"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/game"
	"ships3d/client3ds/impact"
)

// A crash into the world's edge: the grid glows around where the ship touched it, sparks fly
// from there, and the camera shakes (placeCamera). See package impact for how each fades.
const (
	// glowWidthShare is a glowing line's width for its distance from the camera: two or three
	// pixels wide wherever it is, as a line the GPU drew would be, rather than a band near the eye.
	glowWidthShare = 0.012
	glowLeastWidth = 0.05
	sparkCount     = 26
	sparkSize      = 1.4
	sparkSpeed     = 45 // world units a second, at most
	sparkSeed      = 5
)

var (
	glowColour  = [3]float32{0.25, 0.85, 1} // the grid's cyan, bright
	sparkColour = [3]float32{1, 0.75, 0.35} // hot metal
)

// crash is the effects' state between frames.
type crash struct {
	glow       *scene.Node
	sparks     []*scene.Node
	velocities []flight.Vector
	sparkedAt  float64 // the crash the sparks belong to (Game.EdgeHitAt)
	random     *rand.Rand
}

func newCrash(space *scene.Scene) *crash {
	effects := &crash{glow: &scene.Node{Name: "edge glow", Hidden: true, Moving: true}, random: rand.New(rand.NewPCG(sparkSeed, sparkSeed))}
	space.Nodes = append(space.Nodes, effects.glow)
	spark := box("spark", sparkSize, sparkSize, sparkSize, sparkColour)
	spark.Material.Name = "spark"
	for range sparkCount {
		node := &scene.Node{Name: "spark", Mesh: spark, Hidden: true, Moving: true}
		effects.sparks = append(effects.sparks, node)
		effects.velocities = append(effects.velocities, flight.Vector{})
		space.Nodes = append(space.Nodes, node)
	}
	return effects
}

// show moves the effects on to now, for the game's last touch of the edge, seen from eye.
func (effects *crash) show(state *game.Game, edges [][2]flight.Vector, eye flight.Vector, now, seconds float64) {
	effects.showGlow(state, edges, eye, now)
	effects.showSparks(state, now, seconds)
}

// showGlow lays the glow out again every frame it shows: its lines' widths follow the camera.
func (effects *crash) showGlow(state *game.Game, edges [][2]flight.Vector, eye flight.Vector, now float64) {
	fade := impact.Fade(now-state.EdgeTouchAt, impact.GlowSeconds)
	effects.glow.Hidden = state.EdgeTouchAt == 0 || fade == 0
	if effects.glow.Hidden {
		return
	}
	if effects.glow.Mesh == nil {
		effects.glow.Mesh = &scene.Mesh{Name: "edge glow", Material: unlit()}
	}
	glowAround(effects.glow.Mesh, state.EdgePoint, eye, edges, float32(fade))
	effects.glow.Transform = graphics.Identity()
}

// glowAround lays out in mesh the grid's lines near point, each end as bright as it is near and
// as wide as its distance from eye asks.
func glowAround(mesh *scene.Mesh, point, eye flight.Vector, edges [][2]flight.Vector, fade float32) {
	mesh.Positions, mesh.Normals, mesh.Colors = mesh.Positions[:0], mesh.Normals[:0], mesh.Colors[:0]
	for _, edge := range edges {
		from, to := impact.Glow(edge[0].Subtract(point).Length()), impact.Glow(edge[1].Subtract(point).Length())
		if from == 0 && to == 0 {
			continue
		}
		// The nearest point of the line to the eye sets its width, so it is thin where it passes.
		width := max(glowLeastWidth, nearestDistance(eye, edge[0], edge[1])*glowWidthShare)
		appendLineShaded(mesh, edge[0], edge[1], width, glowColour, float32(from)*fade, float32(to)*fade)
	}
	mesh.Dirty = true
}

// nearestDistance is how far point is from the segment from to to.
func nearestDistance(point, from, to flight.Vector) float64 {
	path := to.Subtract(from)
	share := 0.0
	if length := path.Dot(path); length > 0 {
		share = max(0, min(1, point.Subtract(from).Dot(path)/length))
	}
	return point.Subtract(from.Add(path.Times(share))).Length()
}

func (effects *crash) showSparks(state *game.Game, now, seconds float64) {
	if state.EdgeHitAt != 0 && state.EdgeHitAt != effects.sparkedAt {
		effects.sparkedAt = state.EdgeHitAt
		effects.throwSparks(state.EdgePoint)
	}
	fade := impact.Fade(now-effects.sparkedAt, impact.SparkSeconds)
	for index, node := range effects.sparks {
		node.Hidden = effects.sparkedAt == 0 || fade == 0
		if node.Hidden {
			continue
		}
		at := node.Transform
		position := flight.Vector{float64(at[12]), float64(at[13]), float64(at[14])}.Add(effects.velocities[index].Times(seconds))
		node.Transform = graphics.Translate(vectorOf(position, 0))
	}
	if len(effects.sparks) > 0 {
		effects.sparks[0].Mesh.Material.BaseColor = graphics.Vec4{float32(fade), float32(fade), float32(fade), 1}
	}
}

// throwSparks starts the sparks at point, flying back into the world and along the edge.
func (effects *crash) throwSparks(point flight.Vector) {
	inward := point.Times(-1 / point.Length())
	for index, node := range effects.sparks {
		scatter := flight.Vector{effects.random.Float64()*2 - 1, effects.random.Float64()*2 - 1, effects.random.Float64()*2 - 1}
		velocity := inward.Times(0.6).Add(scatter)
		effects.velocities[index] = velocity.Times(sparkSpeed * (0.4 + 0.6*effects.random.Float64()) / velocity.Length())
		node.Transform = graphics.Translate(vectorOf(point, 0))
	}
}

// appendLineShaded is appendLine with each end's colour scaled by its own brightness, so a line
// fades along its length.
func appendLineShaded(mesh *scene.Mesh, from, to flight.Vector, width float64, colour [3]float32, fromBrightness, toBrightness float32) {
	start := len(mesh.Colors)
	appendLine(mesh, from, to, width, colour)
	// appendLine lays each strip's corners out as from, from, to, from, to, to (two triangles).
	pattern := [6]bool{false, false, true, false, true, true}
	for vertex := 0; start+vertex*4 < len(mesh.Colors); vertex++ {
		brightness := fromBrightness
		if pattern[vertex%6] {
			brightness = toBrightness
		}
		for channel := range 3 {
			mesh.Colors[start+vertex*4+channel] *= brightness
		}
	}
}
