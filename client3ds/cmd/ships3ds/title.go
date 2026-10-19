//go:build nintendo3ds

package main

import (
	_ "embed"
	"encoding/json"
	"math"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/input"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"
	ui "github.com/akrck02/go-ui"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/protocol"
	"ships3d/client3ds/ship"
)

// titleShipDesign is the Kestrel, the server's built-in fighter (game/defaultShips.go), kept here
// so the title screen needs no server.
//
//go:embed title-ship.json
var titleShipDesign []byte

// The title's flight: the ship crosses the screen from left to right ahead of a still camera,
// rolling as it goes, once every titleLoopSeconds.
const (
	titleLoopSeconds  = 7
	titleCrossing     = 90 // world units from one side to the other
	titleDistance     = 38 // ahead of the camera
	titleHeight       = -4
	titleRollsPerLoop = 1
	titleShipScale    = 1.2
)

// titleCamera looks straight ahead, +Z, from the origin.
var titleCamera = models.Camera{
	Position:    graphics.Vec4{0, 0, 0, 1},
	Target:      graphics.Vec4{0, 0, 1, 1},
	Up:          graphics.Vec4{0, 1, 0, 0},
	Near:        cameraNear,
	Far:         cameraFar,
	FieldOfView: math.Pi / 4,
}

// titleShip builds the title's ship node, hidden until the title shows.
func titleShip() *scene.Node {
	var layers []protocol.Layer
	if err := json.Unmarshal(titleShipDesign, &layers); err != nil {
		println(err.Error()) // the debug log; the title goes on without its ship
	}
	return &scene.Node{Name: "title ship", Mesh: ship.Build(layers, ship.Fine), Hidden: true, Moving: true}
}

// showTitle flies the title's ship at now seconds and places the still camera.
func (world *world) showTitle(now float64, view *models.Camera) {
	*view = titleCamera
	world.view.Eye, world.view.Target = flight.Vector{}, flight.Forward
	share := math.Mod(now, titleLoopSeconds) / titleLoopSeconds
	// Screen right is -X, looking along +Z: the ship goes from +X to -X, its nose toward -X.
	position := flight.Vector{titleCrossing * (0.5 - share), titleHeight + 3*math.Sin(share*2*math.Pi), titleDistance}
	heading := flight.FromEulerXYZ(0, -math.Pi/2, 0)
	roll := flight.FromEulerXYZ(0, 0, share*titleRollsPerLoop*2*math.Pi)
	world.title.Mesh, world.title.Hidden = world.titleMesh, false
	world.title.Transform = transformOf(position, heading.Then(roll), titleShipScale)
	world.own.Hidden, world.reticle.Hidden = true, true
	dustCellsAround(world.dust, flight.Vector{})
	world.sky.Transform = graphics.Identity()
}

// topTitle paints the title's name over the top screen.
type topTitle struct {
	context  *ui.Context
	renderer *ctr.FlatRenderer
	state    *input.State
}

// newTopTitle prepares a go-ui context for the top screen. It has a painter of its own: one
// painter for both screens would overwrite the vertices of one before the GPU drew them.
func newTopTitle(bottomScreen *ctr.BottomScreen, theme ui.Theme) (*topTitle, error) {
	renderer, err := ctr.NewFlatRenderer(bottomScreen)
	if err != nil {
		return nil, err
	}
	context, err := ui.New()
	if err != nil {
		return nil, err
	}
	context.Theme = theme
	state := input.NewState()
	state.FramebufferWidth, state.FramebufferHeight = ctr.ScreenWidth, ctr.ScreenHeight
	state.WindowWidth, state.WindowHeight = ctr.ScreenWidth, ctr.ScreenHeight
	return &topTitle{context: context, renderer: renderer, state: state}, nil
}

// paint lays a heading and a caption out and paints them over the top screen, drawn just before
// in one picture.
func (title *topTitle) paint(screen *ctr.Screen, heading, caption string) {
	title.context.Begin(title.state)
	layout := ui.Layout{Direction: ui.Vertical, Width: ui.Fixed(ctr.ScreenWidth), Height: ui.Fixed(ctr.ScreenHeight), AlignX: ui.Center, Gap: 2,
		Padding: ui.Padding{Top: 28}}
	title.context.Box(layout, func() {
		title.context.Text(heading, ui.TextStyle{Size: titleSize, Weight: draw2d.Bold})
		title.context.Caption(caption)
	})
	title.context.End()
	title.renderer.PaintOver(screen, title.context.Drawing())
}
