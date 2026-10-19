//go:build nintendo3ds

package main

import (
	"math"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/scene"

	"ships3d/client3ds/flight"
)

// The ship screen's turning ship, on the touch screen under its name and arrows: a scene of its
// own, a hangar (hangar.go) lit as space is, drawn by a renderer of its own as the radar is.
const (
	previewRadius         = 7
	previewTurnsPerSecond = 0.12
)

// The camera looks at the ship a little from above, where a design shows best: its top, its
// cockpit and its wings, rather than its belly.
var (
	previewPosition = flight.Vector{0, 0, 22}
	previewCamera   = models.Camera{
		Position: graphics.Vec4{0, 7, -3, 1},
		Target:   graphics.Vec4{0, 1, 22, 1},
		Up:       graphics.Vec4{0, 1, 0, 0},
		Near:     1, Far: 400,
	}
)

// previewScene is the touch screen's turning ship.
type previewScene struct {
	scene    *scene.Scene
	renderer *ctr.SceneRenderer
	ship     *scene.Node
	hangar   *hangar
}

func newPreviewScene(screen *ctr.Screen) (*previewScene, error) {
	renderer, err := ctr.NewSceneRenderer(screen)
	if err != nil {
		return nil, err
	}
	stage := scene.New()
	lightSpace(&stage.Environment)
	// The hangar's far dark, under the arrows and the name painted over it.
	stage.Environment.SkyHorizon, stage.Environment.FogColor = hangarBack, hangarBack
	built := newHangar()
	ship := &scene.Node{Name: "preview", Hidden: true, Moving: true}
	stage.Nodes = append(append(stage.Nodes, built.nodes...), ship)
	return &previewScene{scene: stage, renderer: renderer, ship: ship, hangar: built}, nil
}

// paint turns the ship at now seconds and draws it on the touch screen, in a frame begun with
// ctr.BeginFrame; with no mesh (a ship the server hasn't described) only the background.
func (preview *previewScene) paint(surface ctr.Surface, mesh *scene.Mesh, scale, now float64) {
	preview.hangar.show(now)
	preview.ship.Hidden = mesh == nil
	if mesh != nil {
		turn := flight.FromEulerXYZ(0, now*previewTurnsPerSecond*2*math.Pi, 0)
		preview.ship.Mesh = mesh
		preview.ship.Transform = transformOf(previewPosition, turn, scale)
	}
	preview.renderer.PaintOn(surface, preview.scene, previewCamera)
}
