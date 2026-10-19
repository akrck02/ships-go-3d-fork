//go:build nintendo3ds

package main

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/protocol"
	"ships3d/client3ds/ship"
)

// The hangar the ship screen's ship turns in: a round platform floating in open space, with a
// glowing rim, a planet far behind and spotlights round its edge whose beams sweep up into the
// stars. It is built of the same parts as the players' ships, lit as they are; the beams are
// translucent. Distances are in the preview's units; the ship is previewRadius big at
// previewPosition.
const (
	platformTop       = -2.6
	platformRadius    = 10
	platformThickness = 1.2
	keelDepth         = 7
	spotlightsAround  = 4
	spotlightRadius   = platformRadius - 0.8
	lampSize          = 1
	beamHeight        = 40
	beamSlices        = 6
	beamWidthAtBase   = 0.6
	beamWidthAtTop    = 3
	beamOpacityAtBase = 0.1
	beamLean          = 0.3 // radians the beams lean outward
	beamSway          = 0.2 // radians they sweep either way
	beamSwaySpeed     = 0.5 // radians a second
	previewSkyRadius  = 150 // the stars' sphere, inside the preview camera's reach
	planetDiameter    = 45
)

// The hangar's colours: the parts' as the editor writes them, the beams' and the dark's linear.
var (
	hangarHull   = "#3a4658"
	hangarDeck   = "#5a6a80"
	hangarKeel   = "#252d3a"
	hangarRim    = "#5fb0ff"
	hangarLamp   = "#e8f2ff"
	hangarPlanet = "#6a4fb0"
	hangarBeam   = [3]float32{0.45, 0.7, 1.0}
	hangarBack   = graphics.Vec4{0.002, 0.004, 0.012, 1} // space, what the screen is cleared to
	planetPlace  = flight.Vector{-70, -20, 200}
)

// hangar is the hangar's nodes: the still part and the spotlights' beams, which sweep.
type hangar struct {
	nodes []*scene.Node
	beams []spotlight
}

// spotlight is one beam, where it stands and which way it leans out.
type spotlight struct {
	node     *scene.Node
	position flight.Vector
	outward  float64 // -1 when it stands left of the ship, 1 right
	phase    float64 // where in its sweep it starts, radians
}

func newHangar() *hangar {
	built := &hangar{}
	shrink := float32(previewSkyRadius) / skyRadius
	stars := &scene.Node{Name: "stars", Mesh: stars(), Transform: graphics.Scale(graphics.Vec4{shrink, shrink, shrink, 0})}
	still := &scene.Node{Name: "hangar", Transform: graphics.Identity(), Mesh: ship.Build([]protocol.Layer{{Visible: true, Parts: hangarParts()}}, ship.Fine)}
	built.nodes = append(built.nodes, stars, still)
	beam := beamMesh()
	for index := range spotlightsAround {
		position := spotlightPlace(index)
		node := &scene.Node{Name: "beam", Mesh: beam}
		built.nodes = append(built.nodes, node)
		built.beams = append(built.beams, spotlight{node: node, position: position, outward: math.Copysign(1, position[0]), phase: float64(index) * 1.7})
	}
	return built
}

// spotlightPlace is where a spotlight stands on the platform's edge, the four of them between
// the ship's nose, tail and sides.
func spotlightPlace(index int) flight.Vector {
	angle := (float64(index) + 0.5) * 2 * math.Pi / spotlightsAround
	return flight.Vector{spotlightRadius * math.Sin(angle), platformTop, previewPosition[2] + spotlightRadius*math.Cos(angle)}
}

// hangarParts are the platform, its rim, its keel, the lamps and the far planet.
func hangarParts() []protocol.Part {
	centre := previewPosition[2]
	part := func(kind string, position, rotation, scale flight.Vector, colour string) protocol.Part {
		return protocol.Part{Type: kind, Position: position, Rotation: rotation, Scale: scale, Color: colour}
	}
	upsideDown := flight.Vector{math.Pi, 0, 0}
	parts := []protocol.Part{
		part("cylinder", flight.Vector{0, platformTop - platformThickness/2, centre}, flight.Vector{}, flight.Vector{2 * platformRadius, platformThickness, 2 * platformRadius}, hangarHull),
		part("cylinder", flight.Vector{0, platformTop + 0.05, centre}, flight.Vector{}, flight.Vector{2*platformRadius - 3, 0.1, 2*platformRadius - 3}, hangarDeck),
		part("torus", flight.Vector{0, platformTop, centre}, flight.Vector{math.Pi / 2, 0, 0}, flight.Vector{2*platformRadius + 0.6, 2*platformRadius + 0.6, 0.6}, hangarRim),
		part("cone", flight.Vector{0, platformTop - platformThickness - keelDepth/2, centre}, upsideDown, flight.Vector{1.4 * platformRadius, keelDepth, 1.4 * platformRadius}, hangarKeel),
		part("sphere", planetPlace, flight.Vector{}, flight.Vector{planetDiameter, planetDiameter, planetDiameter}, hangarPlanet),
	}
	for index := range spotlightsAround {
		lamp := spotlightPlace(index)
		lamp[1] += lampSize / 4
		parts = append(parts, part("sphere", lamp, flight.Vector{}, flight.Vector{lampSize, lampSize / 2, lampSize}, hangarLamp))
	}
	return parts
}

// show sweeps the beams to where they are at now seconds.
func (built *hangar) show(now float64) {
	for _, light := range built.beams {
		lean := -light.outward * (beamLean + beamSway*math.Sin(now*beamSwaySpeed+light.phase))
		light.node.Transform = transformOf(light.position, flight.FromEulerXYZ(0, 0, lean), 1)
	}
}

// beamMesh is a spotlight's beam, standing up from its foot: slices that widen and fade as they
// rise, so the light seems to spread into the dark.
func beamMesh() *scene.Mesh {
	mesh := &scene.Mesh{Name: "beam", Material: &scene.Material{Kind: scene.KindUnlit, BaseColor: graphics.Vec4{1, 1, 1, 1}, Transparent: true, DoubleSided: true}}
	sliceHeight := float32(beamHeight) / beamSlices
	for slice := range beamSlices {
		share := (float32(slice) + 0.5) / beamSlices
		width := beamWidthAtBase + (beamWidthAtTop-beamWidthAtBase)*share
		start := len(mesh.Colors)
		appendBox(mesh, [3]float32{0, (float32(slice) + 0.5) * sliceHeight, 0}, width, sliceHeight, width, hangarBeam)
		for colour := start + 3; colour < len(mesh.Colors); colour += 4 {
			mesh.Colors[colour] = beamOpacityAtBase * (1 - share)
		}
	}
	return mesh
}
