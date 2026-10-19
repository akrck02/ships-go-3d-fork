package ship

import (
	"math"
	"strconv"

	"github.com/akrck02/go-renderer/scene"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/protocol"
)

// shipMaterial lights every ship like the sun lights the island: the browser's flat shading.
var shipMaterial = &scene.Material{Name: "ship", Kind: scene.KindLit, BaseColor: [4]float32{1, 1, 1, 1}, DoubleSided: true}

// Build makes one mesh of a ship's visible parts at a detail, in the design's own units: the
// caller scales it to the game's ship size. Parts of a type it doesn't know are left out.
func Build(layers []protocol.Layer, detail Detail) *scene.Mesh {
	mesh := &scene.Mesh{Name: "ship", Material: shipMaterial}
	// Made to measure: growing them by appending would hold up to twice the memory for a moment,
	// which a design of hundreds of parts can't spare on the console.
	vertices := 3 * Triangles(layers, detail)
	mesh.Positions = make([]float32, 0, 3*vertices)
	mesh.Normals = make([]float32, 0, 3*vertices)
	mesh.Colors = make([]float32, 0, 4*vertices)
	for _, layer := range layers {
		if !layer.Visible {
			continue
		}
		for _, part := range layer.Parts {
			addPart(mesh, part, detail, part.Position, part.Rotation)
			if part.Mirror {
				// The browser's mirror: across X = 0, which turns the other way about Y and Z.
				addPart(mesh, part, detail, flight.Vector{-part.Position[0], part.Position[1], part.Position[2]},
					flight.Vector{part.Rotation[0], -part.Rotation[1], -part.Rotation[2]})
			}
		}
	}
	return mesh
}

// Triangles is how many triangles Build makes of a design at a detail.
func Triangles(layers []protocol.Layer, detail Detail) int {
	count := 0
	for _, layer := range layers {
		if !layer.Visible {
			continue
		}
		for _, part := range layer.Parts {
			triangles := len(primitive(part.Type, detail))
			if part.Mirror {
				triangles *= 2
			}
			count += triangles
		}
	}
	return count
}

// Within is the finest detail at which a design has at most maximum triangles, or the plainest
// when even that has more.
func Within(layers []protocol.Layer, maximum int) Detail {
	for _, detail := range Levels {
		if Triangles(layers, detail) <= maximum {
			return detail
		}
	}
	return Levels[len(Levels)-1]
}

// addPart places a primitive's triangles with the part's scale, then rotation, then position.
func addPart(mesh *scene.Mesh, part protocol.Part, detail Detail, position, rotation flight.Vector) {
	turn := flight.FromEulerXYZ(rotation[0], rotation[1], rotation[2])
	colour := linearColour(part.Color)
	for _, corners := range primitive(part.Type, detail) {
		var placed triangle
		for index, corner := range corners {
			scaled := flight.Vector{corner[0] * part.Scale[0], corner[1] * part.Scale[1], corner[2] * part.Scale[2]}
			placed[index] = point(turn.Apply(scaled).Add(position))
		}
		normal := faceNormal(placed)
		for _, corner := range placed {
			mesh.Positions = append(mesh.Positions, float32(corner[0]), float32(corner[1]), float32(corner[2]))
			mesh.Normals = append(mesh.Normals, normal[0], normal[1], normal[2])
			mesh.Colors = append(mesh.Colors, colour[0], colour[1], colour[2], 1)
		}
	}
}

// faceNormal is the unit vector out of a triangle's face, so each face is lit flat.
func faceNormal(corners triangle) [3]float32 {
	first := flight.Vector(corners[1]).Subtract(flight.Vector(corners[0]))
	second := flight.Vector(corners[2]).Subtract(flight.Vector(corners[0]))
	cross := flight.Vector{
		first[1]*second[2] - first[2]*second[1],
		first[2]*second[0] - first[0]*second[2],
		first[0]*second[1] - first[1]*second[0],
	}
	length := cross.Length()
	if length == 0 {
		return [3]float32{0, 1, 0} // a triangle squashed to a line: any normal will do
	}
	return [3]float32{float32(cross[0] / length), float32(cross[1] / length), float32(cross[2] / length)}
}

// fallbackColour is a part's grey when its colour can't be read.
const fallbackColour = "#9aa7b8"

// linearColour reads "#rrggbb" in the screen's sRGB and returns its linear colour, as three.js
// does with a material's colour.
func linearColour(hex string) [3]float32 {
	value, err := strconv.ParseUint(trimHash(hex), 16, 32)
	if err != nil || len(hex) != len(fallbackColour) {
		value, _ = strconv.ParseUint(fallbackColour[1:], 16, 32) // a constant that always reads
	}
	var colour [3]float32
	for index := range colour {
		channel := float64((value>>(16-8*index))&0xFF) / 0xFF
		colour[index] = float32(sRGBToLinear(channel))
	}
	return colour
}

func trimHash(hex string) string {
	if len(hex) > 0 && hex[0] == '#' {
		return hex[1:]
	}
	return hex
}

// sRGBToLinear is the sRGB transfer curve undone.
func sRGBToLinear(channel float64) float64 {
	if channel < 0.04045 {
		return channel / 12.92
	}
	return math.Pow((channel+0.055)/1.055, 2.4)
}

// Radius is how far the ship reaches from its centre, as the server measures it
// (models.Project3d.BoundingRadius), which sets its scale in the game.
func Radius(layers []protocol.Layer) float64 {
	radius := 0.0
	for _, layer := range layers {
		if !layer.Visible {
			continue
		}
		for _, part := range layer.Parts {
			radius = max(radius, part.Position.Length()+part.Scale.Length()/2)
		}
	}
	return radius
}
