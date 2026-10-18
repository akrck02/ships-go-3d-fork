//go:build nintendo3ds

package main

import (
	"math"
	"strconv"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"

	"ships3d/client3ds/radar"
)

// The map's look, in pixels of the touch screen.
const (
	mapMargin         = 14
	blipSize          = 8
	ownMarkerSize     = 10
	sweepDots         = 40
	sweepDotSize      = 2
	sweepTrailRadians = 0.06
	stalkWidth        = 2
	ringWidth         = 1.5
	heightShare       = 0.6 // how much of the map's radius a ship at the edge, straight above, rises
	labelSize         = 13
	labelMargin       = 8
	labelLineSpacing  = 16
)

var (
	mapRing         = draw2d.Hex("#2c4a5a")
	mapRingFaint    = draw2d.Hex("#1d323d")
	mapOwn          = draw2d.Hex("#4fc3f7")
	mapEnemy        = draw2d.Hex("#ff5252")
	mapEnemyFar     = draw2d.Hex("#8a3434")
	mapSweep        = draw2d.Hex("#4fc3f7")
	sweepTrail      = []float32{0.9, 0.45, 0.2}
	mapLabel        = draw2d.Hex("#9fb3c0")
	mapTransparent  = draw2d.Color{}
	mapCircleRadius = float32(1 << 10) // rounding larger than the shape makes a circle
)

// buildMap fills the touch screen with the map of the ships around ours.
func (session *session) buildMap(context *ui.Context) {
	layout := ui.Layout{Width: ui.Fixed(ctr.BottomWidth), Height: ui.Fixed(ctr.BottomHeight)}
	context.Paint(layout, session.paintMap)
}

func (session *session) paintMap(area draw2d.Rect, drawing *draw2d.List) {
	state := session.game
	distance := radar.Ranges[session.mapRange]
	radius := min(area.Width, area.Height)/2 - mapMargin
	centreX, centreY := area.X+area.Width/2, area.Y+area.Height/2
	circle := func(x, y, size float32, fill, border draw2d.Color, borderWidth float32) {
		drawing.Rectangle(draw2d.Rect{X: x - size/2, Y: y - size/2, Width: size, Height: size}, mapCircleRadius, fill, border, borderWidth)
	}
	circle(centreX, centreY, radius*2, mapTransparent, mapRing, ringWidth)
	circle(centreX, centreY, radius, mapTransparent, mapRingFaint, ringWidth)
	drawing.Rectangle(draw2d.Rect{X: centreX - ringWidth/2, Y: centreY - radius, Width: ringWidth, Height: radius * 2}, 0, mapRingFaint, mapTransparent, 0)
	drawing.Rectangle(draw2d.Rect{X: centreX - radius, Y: centreY - ringWidth/2, Width: radius * 2, Height: ringWidth}, 0, mapRingFaint, mapTransparent, 0)

	toScreen := func(blip radar.Blip) (x, y, top float32) {
		x = centreX + float32(blip.Right)*radius
		y = centreY - float32(blip.Ahead)*radius
		return x, y, y - float32(blip.Above)*radius*heightShare
	}
	session.paintSweep(drawing, centreX, centreY, radius)
	for _, echo := range session.scanner.Echoes {
		x, y, top := toScreen(echo.Blip)
		colour := mapEnemy
		if !echo.Blip.InRange {
			colour = mapEnemyFar
		}
		colour[3] *= float32(echo.Brightness(session.now))
		// A stalk from the map's plane up (or down) to the ship, as in the browser: above and
		// below read at a glance.
		drawing.Rectangle(draw2d.Rect{X: x - stalkWidth/2, Y: min(y, top), Width: stalkWidth, Height: abs(top - y)}, 0, colour, mapTransparent, 0)
		circle(x, top, blipSize, colour, mapTransparent, 0)
	}
	if state.Alive {
		circle(centreX, centreY, ownMarkerSize, mapOwn, mapTransparent, 0)
	}

	lines := []string{
		lifeLine(state.Player.Life, state.Settings.MaxLife),
		strconv.Itoa(state.Player.Kills) + " kills, " + strconv.Itoa(state.Player.Deaths) + " deaths",
	}
	if !state.Alive {
		lines = append(lines, "Destroyed: A to fly again")
	}
	for index, line := range lines {
		drawing.Text(line, area.X+labelMargin, area.Y+labelMargin+labelSize+float32(index)*labelLineSpacing, labelSize, draw2d.Regular, mapLabel)
	}
	rangeText := strconv.Itoa(int(distance)) + " units  (up/down)"
	drawing.Text(rangeText, area.X+labelMargin, area.Y+area.Height-labelMargin, labelSize, draw2d.Regular, mapLabel)
	drawing.Text("START: options", area.X+area.Width-112, area.Y+area.Height-labelMargin, labelSize, draw2d.Regular, mapLabel)
}

// paintSweep draws the radar's turning line as a row of dots, since the draw list has no lines,
// with a fainter one just behind it.
func (session *session) paintSweep(drawing *draw2d.List, centreX, centreY, radius float32) {
	for trail, brightness := range sweepTrail {
		angle := radar.Angle(session.now) - float64(trail)*sweepTrailRadians
		right, ahead := float32(math.Sin(angle)), float32(math.Cos(angle))
		colour := mapSweep
		colour[3] *= brightness
		for dot := 1; dot <= sweepDots; dot++ {
			share := float32(dot) / sweepDots
			drawing.Rectangle(draw2d.Rect{X: centreX + right*radius*share - sweepDotSize/2, Y: centreY - ahead*radius*share - sweepDotSize/2, Width: sweepDotSize, Height: sweepDotSize}, 0, colour, mapTransparent, 0)
		}
	}
}

func abs(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}
