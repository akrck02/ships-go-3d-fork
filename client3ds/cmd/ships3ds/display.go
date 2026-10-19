//go:build nintendo3ds

package main

import (
	"github.com/akrck02/go-3ds/app"
	"github.com/akrck02/go-3ds/bottom"
	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/models"
	"github.com/akrck02/go-renderer/opengl/overlay"
)

// display is both screens and what paints them: the top screen's scene and its words, the touch
// screen's panel, its radar and its turning ship.
type display struct {
	screen   *ctr.Screen
	measurer draw2d.Measurer
	panel    *bottom.Interface
	renderer *ctr.SceneRenderer
	dish     *radarScene
	preview  *previewScene
	title    *topTitle
	readings draw2d.List // the top screen's readings over the game (hud.go), reused
}

// openDisplay opens the screens and makes their painters, the pixel fonts first, as the
// painters read them when made.
func openDisplay() (*display, error) {
	screen, err := ctr.Open()
	if err != nil {
		return nil, err
	}
	shown := &display{screen: screen}
	usePixelFonts()
	if shown.measurer, err = overlay.NewMeasurer(); err != nil {
		return shown.failed(err)
	}
	if shown.panel, err = bottom.New(spaceTheme()); err != nil {
		return shown.failed(err)
	}
	if shown.renderer, err = ctr.NewSceneRenderer(screen); err != nil {
		return shown.failed(err)
	}
	if shown.dish, err = newRadarScene(screen); err != nil {
		return shown.failed(err)
	}
	if shown.preview, err = newPreviewScene(screen); err != nil {
		return shown.failed(err)
	}
	if shown.title, err = newTopTitle(shown.panel.Screen(), spaceTheme()); err != nil {
		return shown.failed(err)
	}
	return shown, nil
}

// failed closes what was opened and passes the error on.
func (shown *display) failed(err error) (*display, error) {
	shown.close()
	return nil, err
}

func (shown *display) close() { shown.screen.Close() }

// paintTop draws the top screen: a heading over a still picture (the title, the ship being
// chosen, the death screen), or the game in 3D with its readings over it.
func (shown *display) paintTop(session *session, world *world, view models.Camera, now float64) {
	if heading, caption := session.topHeading(); heading != "" {
		// One picture rather than two eyes: the words painted over it would show double in 3D.
		shown.renderer.PaintOn(shown.screen, world.scene, view)
		shown.title.paint(shown.screen, heading, caption)
		return
	}
	separation := ctr.Slider3D() * eyeSeparation
	shown.renderer.PaintStereo(world.scene, view, separation, focalDistance)
	if session.screen == gameScreen && session.game.Welcomed {
		paintHUD(&shown.readings, session.game, now)
		shown.title.paintOver(shown.screen, &shown.readings, separation > 0)
	}
}

// paintBottom draws the touch screen: the 3D radar or the turning ship under the panel's
// painting, or the panel alone.
func (shown *display) paintBottom(session *session, world *world, frame app.Frame, now float64) {
	switch {
	case session.showsRadar3D():
		shown.dish.paint(shown.panel.Screen(), session.scanner, now)
		shown.panel.PaintOver(frame.Input, session.build)
	case session.screen == shipScreen && !session.menuOpen && session.confirming == noChange:
		mesh, scale := session.previewShip(world)
		shown.preview.paint(shown.panel.Screen(), mesh, scale, now)
		shown.panel.PaintOver(frame.Input, session.build)
	default:
		shown.panel.Paint(frame.Input, session.build)
	}
}
