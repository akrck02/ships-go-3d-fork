//go:build nintendo3ds

package main

import (
	"math"
	"math/rand/v2"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/protocol"
	"ships3d/client3ds/ship"
)

// The space around the ships, as the browser's (ships-vue-3d, src/game/world.ts): a near-black
// background, stars and three planets on a sky that follows the camera (so they never get
// closer, and steer by), the world's edge as a faint grid, and dust that keeps around the ship to
// show its speed. Distances are the browser's: the planets, at 4,800, stay behind every part of
// the world's edge (a sphere of 1,500, never more than 3,000 away), so its lines cross over them.
const (
	skyRadius        = 6000
	starCount        = 1500
	starSize         = 22 // world units at skyRadius: about a pixel and a half, as the browser's
	starSeed         = 11
	planetDistance   = skyRadius * 0.8
	edgeWidthShare   = 0.002 // three world units at the edge's far side, 1,500 away
	edgeLeastWidth   = 0.02
	edgeRebuildGap   = 50  // world units the camera moves before the edge's widths are worked out again
	edgeSubdivisions = 2   // the browser's icosahedron is finer: this keeps it to a few thousand triangles
	dustCell         = 300 // the browser's dust box
	dustPerCell      = 300
	dustSeed         = 7
	dustSize         = 0.6
)

// Linear colours, from the browser's sRGB hex.
var (
	spaceBackground = graphics.Vec4{0.0006, 0.0009, 0.003, 1} // #02030a
	skyLight        = graphics.Vec4{0.63, 0.69, 1, 1}         // the hemisphere light's sky, #cfd9ff
	groundLight     = graphics.Vec4{0.010, 0.005, 0.014, 1}   // and its ground, #1a1020
	sunColour       = graphics.Vec4{1, 0.89, 0.72, 1}         // #fff2dd
	starColour      = [3]float32{1, 1, 1}
	edgeColour      = [3]float32{0.004, 0.027, 0.047} // #4fc3f7 at 5 % opacity over the background
	dustColour      = [3]float32{0.15, 0.19, 0.30}    // #8899bb at 60 %
)

// planets are the browser's three: a direction, a radius on its sky, and a colour.
var planets = []struct {
	direction flight.Vector
	radius    float64
	colour    string
}{
	{flight.Vector{0.6, 0.15, 0.78}, 900, "#d2733c"},
	{flight.Vector{-0.7, -0.35, -0.6}, 500, "#4f7cc9"},
	{flight.Vector{-0.15, 0.95, -0.25}, 260, "#9e9e9e"},
}

// lightSpace sets the light as the browser's: a bluish sky above, a dark ground below, a warm sun.
func lightSpace(environment *scene.Environment) {
	environment.SkyHorizon, environment.FogColor = spaceBackground, spaceBackground // what the screen is cleared to
	environment.SkyZenith = skyLight                                                // the sky's light on the ships
	environment.GroundAmbient = groundLight
	environment.SunColor = sunColour
	environment.SunDirection = graphics.Vec4{1, 0.6, 0.3, 0}.Normalize()
	environment.Ambient = 0.55
}

// skyNode is the stars and the planets, one node the world moves to the camera every frame.
func skyNode() *scene.Node {
	node := &scene.Node{Name: "sky", Transform: graphics.Identity()}
	node.Children = append(node.Children, &scene.Node{Name: "stars", Transform: graphics.Identity(), Mesh: stars()})
	for _, planet := range planets {
		direction := planet.direction.Times(1 / planet.direction.Length())
		radius := planet.radius
		mesh := ship.Build([]protocol.Layer{{Visible: true, Parts: []protocol.Part{{
			Type: "sphere", Scale: flight.Vector{2 * radius, 2 * radius, 2 * radius}, Color: planet.colour}}}}, ship.Fine)
		mesh.Material = &scene.Material{Name: "planet", Kind: scene.KindLit, BaseColor: graphics.Vec4{1, 1, 1, 1}}
		position := direction.Times(planetDistance)
		node.Children = append(node.Children, &scene.Node{Name: "planet", Mesh: mesh,
			Transform: graphics.Translate(graphics.Vec4{float32(position[0]), float32(position[1]), float32(position[2]), 0})})
	}
	return node
}

// stars scatters small diamonds over a sphere of skyRadius.
func stars() *scene.Mesh {
	random := rand.New(rand.NewPCG(starSeed, starSeed))
	mesh := &scene.Mesh{Name: "stars", Material: unlit()}
	for range starCount {
		// A uniform direction: a height from -1 to 1, and an angle around.
		height := random.Float64()*2 - 1
		around := random.Float64() * 2 * math.Pi
		across := math.Sqrt(1 - height*height)
		centre := [3]float32{float32(across * math.Cos(around) * skyRadius), float32(height * skyRadius), float32(across * math.Sin(around) * skyRadius)}
		appendDiamond(mesh, centre, starSize, starColour)
	}
	return mesh
}

// worldEdge is the edge of the world as a faint grid: the edges of a subdivided icosahedron of the
// world's radius, as the browser draws it in wireframe.
// Each line is as wide as its distance from the camera asks, about a pixel, as the browser's
// lines are: a fixed width would be a wide band where the camera passes close to the edge.
func worldEdge(mesh *scene.Mesh, edges [][2]flight.Vector, eye flight.Vector) {
	mesh.Positions, mesh.Normals, mesh.Colors = mesh.Positions[:0], mesh.Normals[:0], mesh.Colors[:0]
	for _, edge := range edges {
		width := max(edgeLeastWidth, nearestDistance(eye, edge[0], edge[1])*edgeWidthShare)
		appendLine(mesh, edge[0], edge[1], width, edgeColour)
	}
	mesh.Dirty = true
}

// edgeLines are the world's edge's lines at its radius.
func edgeLines(radius float64) [][2]flight.Vector {
	edges := icosphereEdges(edgeSubdivisions)
	for index := range edges {
		edges[index] = [2]flight.Vector{edges[index][0].Times(radius), edges[index][1].Times(radius)}
	}
	return edges
}

// dustCellMesh is one cell of dust: dustPerCell specks in a box dustCell wide from the origin.
func dustCellMesh() *scene.Mesh {
	random := rand.New(rand.NewPCG(dustSeed, dustSeed))
	mesh := &scene.Mesh{Name: "dust", Material: unlit()}
	for range dustPerCell {
		centre := [3]float32{random.Float32() * dustCell, random.Float32() * dustCell, random.Float32() * dustCell}
		appendBox(mesh, centre, dustSize, dustSize, dustSize, dustColour)
	}
	return mesh
}

// dustCellsAround places the eight cells of dust nearest a point, so the dust never ends.
func dustCellsAround(nodes []*scene.Node, point flight.Vector) {
	var corner [3]float64
	for axis := range 3 {
		// The cell boundary nearest the point: the eight cells share it as a corner.
		corner[axis] = math.Round(point[axis]/dustCell) * dustCell
	}
	for index, node := range nodes {
		offset := [3]float64{}
		for axis := range 3 {
			offset[axis] = corner[axis] - dustCell
			if index&(1<<axis) != 0 {
				offset[axis] = corner[axis]
			}
		}
		node.Transform = graphics.Translate(graphics.Vec4{float32(offset[0]), float32(offset[1]), float32(offset[2]), 0})
	}
}

func unlit() *scene.Material {
	return &scene.Material{Kind: scene.KindUnlit, BaseColor: graphics.Vec4{1, 1, 1, 1}}
}

// appendDiamond adds an octahedron of size across: a star that looks the same from anywhere.
func appendDiamond(mesh *scene.Mesh, centre [3]float32, size float32, colour [3]float32) {
	half := size / 2
	tips := [6][3]float32{{half, 0, 0}, {-half, 0, 0}, {0, half, 0}, {0, -half, 0}, {0, 0, half}, {0, 0, -half}}
	for _, face := range [8][3]int{{0, 2, 4}, {2, 1, 4}, {1, 3, 4}, {3, 0, 4}, {2, 0, 5}, {1, 2, 5}, {3, 1, 5}, {0, 3, 5}} {
		for _, tip := range face {
			mesh.Positions = append(mesh.Positions, centre[0]+tips[tip][0], centre[1]+tips[tip][1], centre[2]+tips[tip][2])
			mesh.Normals = append(mesh.Normals, 0, 1, 0)
			mesh.Colors = append(mesh.Colors, colour[0], colour[1], colour[2], 1)
		}
	}
}

// appendLine adds a line from from to to as two thin crossed strips, so it shows from any side.
func appendLine(mesh *scene.Mesh, from, to flight.Vector, width float64, colour [3]float32) {
	along := to.Subtract(from)
	// Two directions across the line: one away from the world's centre, and one across both.
	outward := from.Add(to).Times(0.5)
	outward = outward.Times(1 / outward.Length())
	sideways := flight.Vector{along[1]*outward[2] - along[2]*outward[1], along[2]*outward[0] - along[0]*outward[2], along[0]*outward[1] - along[1]*outward[0]}
	sideways = sideways.Times(1 / sideways.Length())
	for _, across := range []flight.Vector{outward.Times(width / 2), sideways.Times(width / 2)} {
		corners := [4]flight.Vector{from.Subtract(across), from.Add(across), to.Add(across), to.Subtract(across)}
		for _, index := range [6]int{0, 1, 2, 0, 2, 3} {
			corner := corners[index]
			mesh.Positions = append(mesh.Positions, float32(corner[0]), float32(corner[1]), float32(corner[2]))
			mesh.Normals = append(mesh.Normals, 0, 1, 0)
			mesh.Colors = append(mesh.Colors, colour[0], colour[1], colour[2], 1)
		}
	}
}

// icosphereEdges are the edges of an icosahedron whose faces were each split into four, as many
// times as subdivisions, on the unit sphere: each edge once.
func icosphereEdges(subdivisions int) [][2]flight.Vector {
	golden := (1 + math.Sqrt(5)) / 2
	points := []flight.Vector{
		{-1, golden, 0}, {1, golden, 0}, {-1, -golden, 0}, {1, -golden, 0},
		{0, -1, golden}, {0, 1, golden}, {0, -1, -golden}, {0, 1, -golden},
		{golden, 0, -1}, {golden, 0, 1}, {-golden, 0, -1}, {-golden, 0, 1},
	}
	faces := [][3]int{
		{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11}, {1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8},
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9}, {4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1},
	}
	for index := range points {
		points[index] = points[index].Times(1 / points[index].Length())
	}
	for range subdivisions {
		middles := map[[2]int]int{}
		middle := func(first, second int) int {
			key := [2]int{min(first, second), max(first, second)}
			if found, ok := middles[key]; ok {
				return found
			}
			point := points[first].Add(points[second])
			points = append(points, point.Times(1/point.Length()))
			middles[key] = len(points) - 1
			return len(points) - 1
		}
		split := make([][3]int, 0, len(faces)*4)
		for _, face := range faces {
			ab, bc, ca := middle(face[0], face[1]), middle(face[1], face[2]), middle(face[2], face[0])
			split = append(split, [3]int{face[0], ab, ca}, [3]int{face[1], bc, ab}, [3]int{face[2], ca, bc}, [3]int{ab, bc, ca})
		}
		faces = split
	}
	seen := map[[2]int]bool{}
	var edges [][2]flight.Vector
	for _, face := range faces {
		for corner := range 3 {
			first, second := face[corner], face[(corner+1)%3]
			key := [2]int{min(first, second), max(first, second)}
			if !seen[key] {
				seen[key] = true
				edges = append(edges, [2]flight.Vector{points[first], points[second]})
			}
		}
	}
	return edges
}
