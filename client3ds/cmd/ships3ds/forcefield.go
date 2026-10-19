//go:build nintendo3ds

package main

import (
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/game"
)

// The force field shows itself as the ship nears the world's edge: the face of the edge nearest
// the ship fills in, more opaque the nearer, though never wholly, so that space still shows
// through.
const (
	fieldMostOpacity = 0.55
	fieldCentreShare = 1.0 // the face's middle, the brightest
	fieldCornerShare = 0.4 // its corners, fainter
	cameraPullBack   = 1.6 // how much farther back the camera draws, at the edge, in its distances
)

// fieldColour is the field's blue, linear.
var fieldColour = [3]float32{0.25, 0.6, 1}

// forceField is the face nearest the ship.
type forceField struct {
	face  *scene.Node
	shown [3]flight.Vector // the face laid out last, not to lay it out again every frame
}

func newForceField(space *scene.Scene) *forceField {
	material := &scene.Material{Kind: scene.KindUnlit, BaseColor: graphics.Vec4{1, 1, 1, 1}, Transparent: true, DoubleSided: true}
	field := &forceField{face: &scene.Node{Name: "force field", Hidden: true, Moving: true, Transform: graphics.Identity(),
		Mesh: &scene.Mesh{Name: "force field", Material: material}}}
	space.Nodes = append(space.Nodes, field.face)
	return field
}

// show fills in the face nearest our ship, as opaque as the ship is near the edge.
func (field *forceField) show(state *game.Game, triangles [][3]flight.Vector) {
	nearness := state.EdgeNearness()
	field.face.Hidden = nearness == 0 || len(triangles) == 0
	if field.face.Hidden {
		return
	}
	if nearest := struckTriangle(triangles, state.Own.Position); nearest != field.shown {
		field.shown = nearest
		fillFace(field.face.Mesh, nearest)
	}
	opacity := float32(nearness * fieldMostOpacity)
	field.face.Mesh.Material.BaseColor = graphics.Vec4{1, 1, 1, opacity}
}

// fillFace lays a face out just inside the edge, its middle brighter than its corners.
func fillFace(mesh *scene.Mesh, triangle [3]flight.Vector) {
	mesh.Positions, mesh.Normals, mesh.Colors = mesh.Positions[:0], mesh.Normals[:0], mesh.Colors[:0]
	middle := triangle[0].Add(triangle[1]).Add(triangle[2]).Times(1.0 / 3)
	for corner := range 3 {
		points := [3]flight.Vector{middle, triangle[corner], triangle[(corner+1)%3]}
		for index, point := range points {
			inside := point.Times(fieldInsetShare)
			share := float32(fieldCornerShare)
			if index == 0 {
				share = fieldCentreShare
			}
			mesh.Positions = append(mesh.Positions, float32(inside[0]), float32(inside[1]), float32(inside[2]))
			mesh.Normals = append(mesh.Normals, 0, 1, 0)
			mesh.Colors = append(mesh.Colors, fieldColour[0], fieldColour[1], fieldColour[2], share)
		}
	}
	mesh.Dirty = true
}
