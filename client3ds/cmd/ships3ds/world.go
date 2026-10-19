//go:build nintendo3ds

package main

import (
	"math/rand/v2"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"

	"ships3d/client3ds/chase"
	"ships3d/client3ds/flight"
	"ships3d/client3ds/game"
	"ships3d/client3ds/protocol"
	"ships3d/client3ds/ship"
)

// The camera, in world units: a chase camera above and behind the ship, as the browser's, in ship
// radii.
const (
	cameraNear    = 0.5
	cameraFar     = 1200
	focalDistance = 15
	eyeSeparation = 0.4 // with the 3D slider at its top
)

// Dust: specks scattered through space, so flying straight still shows movement.
const (
	dustSpecks   = 600
	dustSize     = 0.6
	dustSeed     = 7
	bulletSize   = 0.5
	bulletLength = 3
	maximumShown = 64 // bullets drawn at once
)

var (
	dustColour        = [3]float32{0.7, 0.75, 0.85}
	ownBulletColour   = [3]float32{1, 0.85, 0.3}
	otherBulletColour = [3]float32{1, 0.35, 0.3}
	reticleColour     = [3]float32{0.45, 0.85, 1}
)

// world is what the top screen shows.
type world struct {
	scene                  *scene.Scene
	own                    *scene.Node
	ownShip                string
	ships                  map[string]*scene.Mesh // built once per design
	ownBullet, otherBullet *scene.Mesh
	camera                 chase.Camera
	wasAlive               bool
	reticle                *scene.Node
	remotes                map[string]*scene.Node
	bullets                []*scene.Node
}

func newWorld() *world {
	space := scene.New()
	space.Environment.SkyZenith = graphics.Vec4{0.01, 0.01, 0.04, 1}
	space.Environment.SkyHorizon = graphics.Vec4{0.03, 0.03, 0.08, 1}
	space.Environment.FogColor = space.Environment.SkyHorizon
	space.Environment.Ambient = 0.45
	space.Environment.SunDirection = graphics.Vec4{-0.4, 0.8, 0.45, 0}.Normalize()
	space.Nodes = append(space.Nodes, &scene.Node{Name: "dust", Transform: graphics.Identity(), Mesh: dust()})
	world := &world{scene: space, ships: map[string]*scene.Mesh{}, remotes: map[string]*scene.Node{}}
	world.own = &scene.Node{Name: "own", Hidden: true}
	world.reticle = &scene.Node{Name: "reticle", Mesh: reticle(), Hidden: true}
	space.Nodes = append(space.Nodes, world.own, world.reticle)
	world.ownBullet = box("own bullet", bulletSize, bulletSize, bulletLength, ownBulletColour)
	world.otherBullet = box("other bullet", bulletSize, bulletSize, bulletLength, otherBulletColour)
	for range maximumShown {
		node := &scene.Node{Name: "bullet", Mesh: ownCopy(world.ownBullet), Hidden: true}
		world.bullets = append(world.bullets, node)
		space.Nodes = append(space.Nodes, node)
	}
	return world
}

// show places everything where the game says, and the camera behind our ship.
func (world *world) show(state *game.Game, view *models.Camera, seconds float64) {
	if !state.Welcomed {
		view.Position, view.Target, view.Up = graphics.Vec4{0, 20, -60, 1}, graphics.Vec4{0, 0, 0, 1}, graphics.Vec4{0, 1, 0, 0}
		return
	}
	scale := state.Player.Scale
	if world.ownShip != state.Player.Ship.Id {
		world.ownShip = state.Player.Ship.Id
		world.own.Mesh = ownCopy(world.shipMesh(state.Player.Ship.Id, state.Player.Ship.Layers))
	}
	world.own.Hidden = !state.Alive
	place(world.own, transformOf(state.Own.Position, state.Own.Rotation, scale))
	world.showRemotes(state)
	world.showBullets(state)
	world.placeReticle(state)
	if state.Alive && !world.wasAlive {
		world.camera.Reset() // back from a respawn, far from the wreck
	}
	world.wasAlive = state.Alive
	if state.Alive {
		world.placeCamera(state, view, seconds) // dead, the camera watches the wreck from where it was
	}
}

func (world *world) shipMesh(id string, layers []protocol.Layer) *scene.Mesh {
	if mesh, found := world.ships[id]; found {
		return mesh
	}
	mesh := ship.Build(layers)
	world.ships[id] = mesh
	return mesh
}

func (world *world) showRemotes(state *game.Game) {
	for id, node := range world.remotes {
		if _, found := state.Remotes[id]; !found {
			node.Hidden = true
			delete(world.remotes, id)
		}
	}
	for id, remote := range state.Remotes {
		node, found := world.remotes[id]
		if !found {
			node = world.reuseOrAddNode(world.shipMesh(remote.Player.Ship.Id, remote.Player.Ship.Layers))
			world.remotes[id] = node
		}
		node.Hidden = remote.Player.Dead
		place(node, transformOf(remote.Shown.Position, remote.Shown.Rotation, remote.Player.Scale))
	}
}

// reuseOrAddNode takes a hidden ship node no remote uses any more, or adds one: nodes are never
// removed from the scene, so the renderer keeps what it uploaded.
func (world *world) reuseOrAddNode(mesh *scene.Mesh) *scene.Node {
	for _, node := range world.scene.Nodes {
		if node.Name == "remote" && node.Hidden && !world.inUse(node) {
			node.Mesh = ownCopy(mesh)
			return node
		}
	}
	node := &scene.Node{Name: "remote", Mesh: ownCopy(mesh)}
	world.scene.Nodes = append(world.scene.Nodes, node)
	return node
}

func (world *world) inUse(candidate *scene.Node) bool {
	for _, node := range world.remotes {
		if node == candidate {
			return true
		}
	}
	return false
}

func (world *world) showBullets(state *game.Game) {
	shown := 0
	place := func(bullets []game.Bullet, mesh *scene.Mesh) {
		for _, bullet := range bullets {
			if shown == len(world.bullets) {
				return
			}
			node := world.bullets[shown]
			node.Hidden = false
			if node.Mesh.Colors[0] != mesh.Colors[0] { // the other colour: same shape
				node.Mesh.Colors, node.Mesh.Dirty = mesh.Colors, true
			}
			place(node, transformOf(bullet.Position, rotationAlong(bullet.Direction), 1))
			shown++
		}
	}
	place(state.Bullets, world.ownBullet)
	place(state.Others, world.otherBullet)
	for _, node := range world.bullets[shown:] {
		node.Hidden = true
	}
}

func (world *world) placeCamera(state *game.Game, view *models.Camera, seconds float64) {
	chased := world.camera.Follow(state.Own, state.Settings.ShipRadius, seconds)
	view.Position = vectorOf(chased.Eye, 1)
	view.Target = vectorOf(chased.Target, 1)
	view.Up = vectorOf(chased.Up, 0)
}

// reticleDistance is how far ahead of the ship the aim mark floats, in world units: where a
// bullet is after a fifth of a second.
const reticleDistance = 60

// placeReticle floats the aim mark ahead of our nose, facing us.
func (world *world) placeReticle(state *game.Game) {
	world.reticle.Hidden = !state.Alive
	ahead := state.Own.Position.Add(state.Own.Rotation.Apply(flight.Forward).Times(reticleDistance))
	place(world.reticle, transformOf(ahead, state.Own.Rotation, 1))
}

// reticle is a cross with a gap in the middle, drawn in the plane facing the ship.
func reticle() *scene.Mesh {
	mesh := &scene.Mesh{Name: "reticle", Material: &scene.Material{Kind: scene.KindUnlit, BaseColor: graphics.Vec4{1, 1, 1, 1}}}
	const arm, thickness, gap float32 = 2.5, 0.35, 1.2
	offset := gap + arm/2
	for _, centre := range [4][3]float32{{offset, 0, 0}, {-offset, 0, 0}, {0, offset, 0}, {0, -offset, 0}} {
		width, height := arm, thickness
		if centre[0] == 0 {
			width, height = thickness, arm
		}
		appendBox(mesh, centre, width, height, thickness, reticleColour)
	}
	return mesh
}

func vectorOf(vector flight.Vector, w float32) graphics.Vec4 {
	return graphics.Vec4{float32(vector[0]), float32(vector[1]), float32(vector[2]), w}
}

// ownCopy is a mesh of its own for one node, sharing the vertices. go-renderer's 3DS backend
// bakes the light into a mesh once for every way it is turned and keeps each of those until the
// mesh stops being drawn; a ship that keeps turning would fill the GPU's memory with them. With a
// mesh per node, place can drop the old bake whenever the node turns.
// TODO(akrck02): let ctr keep a few turns per mesh and reuse the oldest, and drop this.
func ownCopy(mesh *scene.Mesh) *scene.Mesh {
	copied := *mesh
	copied.GPU = nil
	return &copied
}

// place moves a node, and has its mesh baked again when it turned (see ownCopy).
func place(node *scene.Node, transform graphics.Mat4) {
	for _, index := range [9]int{0, 1, 2, 4, 5, 6, 8, 9, 10} {
		if node.Transform[index] != transform[index] {
			node.Mesh.Dirty = true
			break
		}
	}
	node.Transform = transform
}

func transformOf(position flight.Vector, rotation flight.Rotation, scale float64) graphics.Mat4 {
	return graphics.FromTranslationRotationScale(
		graphics.Vec4{float32(position[0]), float32(position[1]), float32(position[2]), 1},
		graphics.Vec4{float32(rotation[0]), float32(rotation[1]), float32(rotation[2]), float32(rotation[3])},
		graphics.Vec4{float32(scale), float32(scale), float32(scale), 0},
	)
}

// rotationAlong is a rotation that points +Z along direction, a unit vector.
func rotationAlong(direction flight.Vector) flight.Rotation {
	// Half way between +Z and the direction, as a quaternion: the cross product and 1 + the dot.
	cross := flight.Vector{-direction[1], direction[0], 0}
	if direction[2] < -0.9999 {
		return flight.Rotation{0, 1, 0, 0} // straight back: half a turn about Y
	}
	return flight.Rotation{cross[0], cross[1], cross[2], 1 + direction[2]}.Normalized()
}

// dust scatters small boxes through the world.
func dust() *scene.Mesh {
	random := rand.New(rand.NewPCG(dustSeed, dustSeed))
	mesh := &scene.Mesh{Name: "dust", Material: &scene.Material{Kind: scene.KindUnlit, BaseColor: graphics.Vec4{1, 1, 1, 1}}}
	const spread = 600
	for range dustSpecks {
		centre := [3]float32{(random.Float32()*2 - 1) * spread, (random.Float32()*2 - 1) * spread, (random.Float32()*2 - 1) * spread}
		appendBox(mesh, centre, dustSize, dustSize, dustSize, dustColour)
	}
	return mesh
}

func box(name string, width, height, depth float32, colour [3]float32) *scene.Mesh {
	mesh := &scene.Mesh{Name: name, Material: &scene.Material{Kind: scene.KindUnlit, BaseColor: graphics.Vec4{1, 1, 1, 1}}}
	appendBox(mesh, [3]float32{}, width, height, depth, colour)
	return mesh
}

// appendBox adds a box's twelve triangles to a mesh.
func appendBox(mesh *scene.Mesh, centre [3]float32, width, height, depth float32, colour [3]float32) {
	half := [3]float32{width / 2, height / 2, depth / 2}
	corner := func(x, y, z float32) [3]float32 {
		return [3]float32{centre[0] + x*half[0], centre[1] + y*half[1], centre[2] + z*half[2]}
	}
	faces := [6][4][3]float32{
		{corner(-1, -1, 1), corner(1, -1, 1), corner(1, 1, 1), corner(-1, 1, 1)},
		{corner(1, -1, -1), corner(-1, -1, -1), corner(-1, 1, -1), corner(1, 1, -1)},
		{corner(1, -1, 1), corner(1, -1, -1), corner(1, 1, -1), corner(1, 1, 1)},
		{corner(-1, -1, -1), corner(-1, -1, 1), corner(-1, 1, 1), corner(-1, 1, -1)},
		{corner(-1, 1, 1), corner(1, 1, 1), corner(1, 1, -1), corner(-1, 1, -1)},
		{corner(-1, -1, -1), corner(1, -1, -1), corner(1, -1, 1), corner(-1, -1, 1)},
	}
	for _, face := range faces {
		for _, index := range [6]int{0, 1, 2, 0, 2, 3} {
			mesh.Positions = append(mesh.Positions, face[index][0], face[index][1], face[index][2])
			mesh.Normals = append(mesh.Normals, 0, 1, 0)
			mesh.Colors = append(mesh.Colors, colour[0], colour[1], colour[2], 1)
		}
	}
}
