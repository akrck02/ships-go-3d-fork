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
	speedBarWidth     = 6
	speedBarMargin    = 40
	hitFlashSeconds   = 0.5
	hitFlashWidth     = 6
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
	mapHit          = draw2d.Hex("#ff5252")
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
	radius := min(area.Width, area.Height)/2 - mapMargin
	centreX, centreY := area.X+area.Width/2, area.Y+area.Height/2
	circle := func(x, y, size float32, fill, border draw2d.Color, borderWidth float32) {
		drawing.Rectangle(draw2d.Rect{X: x - size/2, Y: y - size/2, Width: size, Height: size}, mapCircleRadius, fill, border, borderWidth)
	}
	if !session.profile.Settings.FlatRadar {
		session.paintReadings(area, drawing) // the 3D radar is drawn under, by the radar scene
		return
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
	session.paintReadings(area, drawing)
}

// paintReadings draws what both radars show around the dish: the speed, a hit's flash, life,
// score, the range and how to open the options.
func (session *session) paintReadings(area draw2d.Rect, drawing *draw2d.List) {
	state := session.game
	distance := radar.Ranges[session.mapRange]
	session.paintSpeed(drawing, area, state.Own.Speed, state.Settings.MinSpeed, state.Settings.MaxSpeed)
	if session.now-state.HitAt < hitFlashSeconds && state.HitAt > 0 {
		// Hit: the screen's edge flashes red, fading.
		flash := mapHit
		flash[3] *= float32(1 - (session.now-state.HitAt)/hitFlashSeconds)
		drawing.Rectangle(area, 0, mapTransparent, flash, hitFlashWidth)
	}

	lines := []string{
		lifeLine(state.Player.Life, state.Settings.MaxLife),
		strconv.Itoa(state.Player.Kills) + " kills, " + strconv.Itoa(state.Player.Deaths) + " deaths",
	}
	if !state.Alive {
		lines = append(lines, session.respawnLine())
	}
	for index, line := range lines {
		drawing.Text(line, area.X+labelMargin, area.Y+labelMargin+labelSize+float32(index)*labelLineSpacing, labelSize, draw2d.Regular, mapLabel)
	}
	// The zoom rather than the distance to the edge: the game's distances have no unit a player
	// knows. The widest range is ×1.
	zoom := radar.Ranges[len(radar.Ranges)-1] / distance
	rangeText := "ZOOM x" + strconv.Itoa(int(zoom)) + " UP/DOWN"
	drawing.Text(rangeText, area.X+labelMargin, area.Y+area.Height-labelMargin, labelPixels, draw2d.Mono, mapLabel)
	drawing.Text("START MENU", area.X+area.Width-labelMargin-10*labelPixels, area.Y+area.Height-labelMargin, labelPixels, draw2d.Mono, mapLabel)
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

// paintSpeed draws the speed as a bar up the right edge: empty at the slowest reverse, full at
// the fastest, with a mark where the ship stands still.
func (session *session) paintSpeed(drawing *draw2d.List, area draw2d.Rect, speed, slowest, fastest float64) {
	if fastest <= slowest {
		return
	}
	x := area.X + area.Width - labelMargin - speedBarWidth
	top, bottom := area.Y+speedBarMargin, area.Y+area.Height-speedBarMargin
	height := bottom - top
	share := float32((speed - slowest) / (fastest - slowest))
	stopped := bottom - float32(-slowest/(fastest-slowest))*height
	drawing.Rectangle(draw2d.Rect{X: x, Y: top, Width: speedBarWidth, Height: height}, speedBarWidth/2, mapRingFaint, mapTransparent, 0)
	drawing.Rectangle(draw2d.Rect{X: x, Y: bottom - share*height, Width: speedBarWidth, Height: share * height}, speedBarWidth/2, mapOwn, mapTransparent, 0)
	drawing.Rectangle(draw2d.Rect{X: x - 3, Y: stopped - ringWidth/2, Width: speedBarWidth + 6, Height: ringWidth}, 0, mapLabel, mapTransparent, 0)
	drawing.Text(strconv.Itoa(int(speed)), x-labelSize-14, bottom-share*height+labelSize/2, labelSize, draw2d.Regular, mapLabel)
}

// respawnLine says how long until the player may fly again, then how.
func (session *session) respawnLine() string {
	if session.game.CanRespawn(session.now) {
		return "Destroyed: A to fly again"
	}
	return "Destroyed: back in " + strconv.Itoa(int(math.Ceil(session.game.RespawnIn(session.now)))) + " s"
}

func abs(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}
