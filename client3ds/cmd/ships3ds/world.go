//go:build nintendo3ds

package main

import (
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
	cameraFar     = 2600 // past the sky and the world's edge
	focalDistance = 15
	eyeSeparation = 0.4 // with the 3D slider at its top
)

// Bullets.
const (
	// Bullets are small cubes with no turn: a streak along its path would be baked again each
	// time a node took another bullet, and on the 3DS that is the expensive part of a frame.
	bulletSize   = 1.2
	maximumShown = 64 // bullets drawn at once
)

var (
	ownBulletColour   = [3]float32{1, 0.85, 0.3}
	otherBulletColour = [3]float32{1, 0.35, 0.3}
	reticleColour     = [3]float32{0.45, 0.85, 1}
)

// world is what the top screen shows.
type world struct {
	scene                  *scene.Scene
	own                    *scene.Node
	ownShip                string
	ships                  map[meshKey]*scene.Mesh // built once per design and detail
	ownBullet, otherBullet *scene.Mesh
	camera                 chase.Camera
	wasAlive               bool
	reticle                *scene.Node
	remotes                map[string]*scene.Node
	bullets                []*scene.Node
	view                   chase.View // what the camera saw last, to leave out the ships it can't
	sky, edge              *scene.Node
	dust                   []*scene.Node
}

func newWorld() *world {
	space := scene.New()
	lightSpace(&space.Environment)
	world := &world{scene: space, ships: map[meshKey]*scene.Mesh{}, remotes: map[string]*scene.Node{}}
	world.sky = skyNode()
	world.edge = &scene.Node{Name: "world edge", Transform: graphics.Identity(), Hidden: true}
	space.Nodes = append(space.Nodes, world.sky, world.edge)
	cell := dustCellMesh()
	for range 8 {
		node := &scene.Node{Name: "dust", Mesh: cell, Transform: graphics.Identity()}
		world.dust = append(world.dust, node)
		space.Nodes = append(space.Nodes, node)
	}
	// Ships and the aim cross turn all the time: Moving has go-renderer light them on the GPU
	// rather than bake their light again at every turn.
	world.own = &scene.Node{Name: "own", Hidden: true, Moving: true}
	world.reticle = &scene.Node{Name: "reticle", Mesh: reticle(), Hidden: true, Moving: true}
	space.Nodes = append(space.Nodes, world.own, world.reticle)
	world.ownBullet = box("own bullet", bulletSize, bulletSize, bulletSize, ownBulletColour)
	world.otherBullet = box("other bullet", bulletSize, bulletSize, bulletSize, otherBulletColour)
	for range maximumShown {
		node := &scene.Node{Name: "bullet", Mesh: world.ownBullet, Hidden: true}
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
		world.own.Mesh = world.shipMesh(state.Player.Ship.Id, state.Player.Ship.Layers, false)
	}
	world.own.Hidden = !state.Alive
	world.own.Transform = transformOf(state.Own.Position, state.Own.Rotation, scale)
	if state.Alive && !world.wasAlive {
		world.camera.Reset() // back from a respawn, far from the wreck
	}
	world.wasAlive = state.Alive
	if state.Alive {
		world.placeCamera(state, view, seconds) // dead, the camera watches the wreck from where it was
	}
	world.showRemotes(state)
	world.showBullets(state)
	world.placeReticle(state)
	world.placeSpace(state)
}

// placeSpace keeps the sky around the camera, the dust around the ship, and the world's edge, once
// the game has said how large the world is.
func (world *world) placeSpace(state *game.Game) {
	eye := world.view.Eye
	world.sky.Transform = graphics.Translate(graphics.Vec4{float32(eye[0]), float32(eye[1]), float32(eye[2]), 0})
	dustCellsAround(world.dust, state.Own.Position)
	if world.edge.Mesh == nil && state.Settings.WorldRadius > 0 {
		world.edge.Mesh, world.edge.Hidden = worldEdge(state.Settings.WorldRadius), false
	}
}

// meshKey is a design near or far.
type meshKey struct {
	id  string
	far bool
}

// How many triangles a ship may have, near and far. A design may have hundreds of round parts:
// at the fine detail one reached two hundred thousand triangles, more than the console's memory
// holds, so each design gets the finest detail that fits.
const (
	nearTriangles = 12000
	farTriangles  = 2500
)

// shipMesh is a design's mesh near or far, built the first time it is asked for.
func (world *world) shipMesh(id string, layers []protocol.Layer, far bool) *scene.Mesh {
	key := meshKey{id, far}
	if mesh, found := world.ships[key]; found {
		return mesh
	}
	budget := nearTriangles
	if far {
		budget = farTriangles
	}
	mesh := ship.Build(layers, ship.Within(layers, budget))
	world.ships[key] = mesh
	return mesh
}

// Where ships turn coarse, and where they are too far to draw at all, in world units.
const (
	coarseDistance = 60  // from here a ship is a few dozen pixels wide
	sightDistance  = 700 // farther, a ship is a pixel or two
)

func (world *world) showRemotes(state *game.Game) {
	for id, node := range world.remotes {
		if _, found := state.Remotes[id]; !found {
			node.Hidden = true
			delete(world.remotes, id)
		}
	}
	for id, player := range state.Remotes {
		// Only the ships the camera can see are drawn, and only those get a node: the radar shows
		// the rest.
		visible := !player.Player.Dead && world.view.Sees(player.Shown.Position, state.Settings.ShipRadius, sightDistance)
		node, found := world.remotes[id]
		if !visible {
			if found {
				node.Hidden = true
			}
			continue
		}
		if !found {
			node = world.reuseOrAddNode()
			world.remotes[id] = node
		}
		distance := player.Shown.Position.Subtract(state.Own.Position).Length()
		node.Hidden = false
		node.Mesh = world.shipMesh(player.Player.Ship.Id, player.Player.Ship.Layers, distance > coarseDistance)
		node.Transform = transformOf(player.Shown.Position, player.Shown.Rotation, player.Player.Scale)
	}
}

// reuseOrAddNode takes a hidden ship node no remote uses any more, or adds one: nodes are never
// removed from the scene, so the renderer keeps what it uploaded.
func (world *world) reuseOrAddNode() *scene.Node {
	for _, node := range world.scene.Nodes {
		if node.Name == "remote" && node.Hidden && !world.inUse(node) {
			return node
		}
	}
	node := &scene.Node{Name: "remote", Moving: true}
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
	show := func(bullets []game.Bullet, mesh *scene.Mesh) {
		for _, bullet := range bullets {
			if shown == len(world.bullets) {
				return
			}
			node := world.bullets[shown]
			node.Hidden, node.Mesh = false, mesh
			node.Transform = transformOf(bullet.Position, flight.NoRotation, 1)
			shown++
		}
	}
	show(state.Bullets, world.ownBullet)
	show(state.Others, world.otherBullet)
	for _, node := range world.bullets[shown:] {
		node.Hidden = true
	}
}

func (world *world) placeCamera(state *game.Game, view *models.Camera, seconds float64) {
	chased := world.camera.Follow(state.Own, state.Settings.ShipRadius, seconds)
	world.view = chased
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
	world.reticle.Transform = transformOf(ahead, state.Own.Rotation, 1)
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

func transformOf(position flight.Vector, rotation flight.Rotation, scale float64) graphics.Mat4 {
	return graphics.FromTranslationRotationScale(
		graphics.Vec4{float32(position[0]), float32(position[1]), float32(position[2]), 1},
		graphics.Vec4{float32(rotation[0]), float32(rotation[1]), float32(rotation[2]), float32(rotation[3])},
		graphics.Vec4{float32(scale), float32(scale), float32(scale), 0},
	)
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
