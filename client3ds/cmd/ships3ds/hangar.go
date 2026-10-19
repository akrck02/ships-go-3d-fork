//go:build nintendo3ds

package main

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// The hangar the ship screen's ship turns in: a dark deck with a grid, a turntable lit from
// above with a glowing rim, a back wall of light panels over hazard stripes, and two rows of
// lane lights that chase toward the wall. Everything is unlit boxes, its colours its own light.
// Distances are in the preview's units; the ship is previewRadius big at previewPosition.
const (
	deckHeight        = -3.2
	deckHalfWidth     = 34
	deckNear          = 4
	deckFar           = 48
	gridSpacing       = 4
	gridLineWidth     = 0.12
	turntableRadius   = 9
	turntableHeight   = 0.6
	rimWidth          = 0.5
	poolRadius        = 13
	wallDistance      = 46
	wallHeight        = 30
	panelWidth        = 5
	panelHeight       = 3
	panelGap          = 2
	stripeWidth       = 2
	stripeHeight      = 1.6
	laneLightSize     = 0.7
	laneLightSpacing  = 4
	laneLightSide     = 11
	chaseLightsLit    = 3   // how many lights of each row are lit at once
	chaseStepsASecond = 6.0 // how fast the lit ones run toward the wall
)

// The hangar's colours, linear as the renderer's.
var (
	hangarDeck      = [3]float32{0.008, 0.012, 0.028}
	hangarGrid      = [3]float32{0.03, 0.05, 0.10}
	hangarPool      = [3]float32{0.02, 0.035, 0.07}
	hangarTurntable = [3]float32{0.025, 0.035, 0.06}
	hangarRim       = [3]float32{0.12, 0.4, 0.85}
	hangarWall      = [3]float32{0.012, 0.018, 0.04}
	hangarPanel     = [3]float32{0.15, 0.3, 0.6}
	hangarHazard    = [3]float32{0.75, 0.5, 0.03}
	hangarLamp      = [3]float32{0.7, 0.85, 1.0}
	hangarBack      = graphics.Vec4{0.004, 0.007, 0.018, 1} // the far dark, what the screen is cleared to
)

// hangar is the hangar's nodes: the still part and the lane lights, lit in turn.
type hangar struct {
	nodes []*scene.Node
	lanes [][]*scene.Node // each row, near to far
}

func newHangar() *hangar {
	built := &hangar{}
	built.nodes = append(built.nodes, &scene.Node{Name: "hangar", Transform: graphics.Identity(), Mesh: hangarMesh()})
	for _, side := range []float32{-laneLightSide, laneLightSide} {
		row := []*scene.Node{}
		for z := float32(deckNear + 2); z < wallDistance; z += laneLightSpacing {
			light := &scene.Node{Name: "lane light", Mesh: box("lane light", laneLightSize, laneLightSize/2, laneLightSize, hangarLamp),
				Transform: graphics.Translate(graphics.Vec4{side, deckHeight + laneLightSize/4, z, 0})}
			row = append(row, light)
			built.nodes = append(built.nodes, light)
		}
		built.lanes = append(built.lanes, row)
	}
	return built
}

// show lights the lane lights that the chase has reached at now seconds.
func (built *hangar) show(now float64) {
	for _, row := range built.lanes {
		head := int(now*chaseStepsASecond) % (len(row) + chaseLightsLit)
		for index, light := range row {
			light.Hidden = index > head || index <= head-chaseLightsLit
		}
	}
}

// hangarMesh is the hangar's still part, in one mesh.
func hangarMesh() *scene.Mesh {
	mesh := &scene.Mesh{Name: "hangar", Material: &scene.Material{Kind: scene.KindUnlit, BaseColor: graphics.Vec4{1, 1, 1, 1}}}
	deckDepth := float32(deckFar - deckNear)
	appendBox(mesh, [3]float32{0, deckHeight - 0.5, deckNear + deckDepth/2}, 2*deckHalfWidth, 1, deckDepth, hangarDeck)
	for x := float32(-deckHalfWidth); x <= deckHalfWidth; x += gridSpacing {
		appendBox(mesh, [3]float32{x, deckHeight, deckNear + deckDepth/2}, gridLineWidth, 0.02, deckDepth, hangarGrid)
	}
	for z := float32(deckNear); z <= deckFar; z += gridSpacing {
		appendBox(mesh, [3]float32{0, deckHeight, z}, 2*deckHalfWidth, 0.02, gridLineWidth, hangarGrid)
	}
	centre := float32(previewPosition[2])
	appendDisc(mesh, centre, deckHeight+0.03, poolRadius, 0.02, hangarPool) // the lamp's light on the deck
	appendDisc(mesh, centre, deckHeight+0.04, turntableRadius+rimWidth, turntableHeight, hangarRim)
	appendDisc(mesh, centre, deckHeight+0.08, turntableRadius, turntableHeight, hangarTurntable)
	appendWall(mesh)
	return mesh
}

// appendDisc adds a round slab, in steps as pixel art draws a circle: slices across, each as
// wide as the circle there.
func appendDisc(mesh *scene.Mesh, centreZ, bottom, radius, height float32, colour [3]float32) {
	const slice = 1
	for z := -radius + slice/2; z < radius; z += slice {
		width := 2 * float32(math.Sqrt(float64(radius*radius-z*z)))
		appendBox(mesh, [3]float32{0, bottom + height/2, centreZ + z}, width, height, slice, colour)
	}
}

// appendWall adds the back wall: rows of light panels over a band of hazard stripes.
func appendWall(mesh *scene.Mesh) {
	appendBox(mesh, [3]float32{0, deckHeight + wallHeight/2, wallDistance}, 2*deckHalfWidth, wallHeight, 1, hangarWall)
	front := float32(wallDistance - 0.6)
	for x := float32(-deckHalfWidth + 1); x < deckHalfWidth; x += 2 * stripeWidth {
		appendBox(mesh, [3]float32{x, deckHeight + stripeHeight/2, front}, stripeWidth, stripeHeight, 0.2, hangarHazard)
	}
	for row := range 3 {
		y := deckHeight + 4 + float32(row)*(panelHeight+panelGap)
		for x := float32(-deckHalfWidth + panelWidth); x < deckHalfWidth-panelWidth/2; x += panelWidth + panelGap {
			appendBox(mesh, [3]float32{x, y, front}, panelWidth, panelHeight, 0.2, hangarPanel)
		}
	}
}
