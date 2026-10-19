//go:build nintendo3ds

package main

import (
	"math"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"

	"ships3d/client3ds/game"
)

// The top screen's readings over the game: life as a bar of charges at the top left, and the
// copilot's word at the bottom when there is something to say. Sizes are in pixels.
const (
	lifeLeft          = 10
	lifeTop           = 10
	lifeMostSegments  = 10
	lifeSegmentWidth  = 9
	lifeSegmentHeight = 12
	lifeSegmentGap    = 2
	lifeLowShare      = 0.3 // below this much life the charges turn red
	copilotHeight     = 40
	copilotMargin     = 8
	copilotFaceSize   = 24
	copilotWarnShare  = 0.35 // how near the edge the copilot speaks up
	copilotBlinks     = 2.0  // the warning's blinks a second
)

// copilotEdgeWarning is what the copilot says as the ship heads out of the arena.
const copilotEdgeWarning = "We're leaving the combat arena, sir!"

// copilotFace is the copilot's face, a little robot: '#' its shell, '+' its eyes.
var copilotFace = []string{
	"....##....",
	"....##....",
	".########.",
	"##......##",
	"#.++..++.#",
	"#.++..++.#",
	"#........#",
	"#..####..#",
	"##......##",
	".########.",
}

// hudColours are the readings' colours: the charges full, low and spent, and the copilot's box.
var hudColours = struct {
	full, low, spent, box, edge, ink, face, eyes draw2d.Color
}{
	full: draw2d.Hex("#5fb0ff"), low: draw2d.Hex("#ff7088"), spent: draw2d.Hex("#1c2a48c0"),
	box: draw2d.Hex("#0e1c42d0"), edge: draw2d.Hex("#9cc8f2"), ink: draw2d.Hex("#eaf4ff"),
	face: draw2d.Hex("#9cc8f2"), eyes: draw2d.Hex("#5ee0ff"),
}

// paintHUD lays the readings out in drawing for this frame, which it empties first.
func paintHUD(drawing *draw2d.List, state *game.Game, now float64) {
	drawing.Reset()
	paintLife(drawing, state.Player.Life, state.Settings.MaxLife)
	if state.EdgeNearness() >= copilotWarnShare {
		paintCopilot(drawing, copilotEdgeWarning, now)
	}
}

// paintLife draws life as charges: one segment for each tenth of the most life, lit as far as
// life reaches, the last one partly.
func paintLife(drawing *draw2d.List, life, most float64) {
	if most <= 0 {
		return
	}
	segments := min(int(math.Ceil(most)), lifeMostSegments)
	share := life / most
	lit := hudColours.full
	if share < lifeLowShare {
		lit = hudColours.low
	}
	drawing.Text("LIFE", lifeLeft, lifeTop+lifeSegmentHeight-2, labelPixels, draw2d.Mono, hudColours.ink)
	left := float32(lifeLeft + 5*labelPixels)
	for segment := range segments {
		x := left + float32(segment*(lifeSegmentWidth+lifeSegmentGap))
		drawing.Rectangle(draw2d.Rect{X: x, Y: lifeTop, Width: lifeSegmentWidth, Height: lifeSegmentHeight}, 0, hudColours.spent, draw2d.Color{}, 0)
		filled := min(max(share*float64(segments)-float64(segment), 0), 1)
		if filled > 0 {
			height := float32(filled) * lifeSegmentHeight
			drawing.Rectangle(draw2d.Rect{X: x, Y: lifeTop + lifeSegmentHeight - height, Width: lifeSegmentWidth, Height: height}, 0, lit, draw2d.Color{}, 0)
		}
	}
}

// paintCopilot draws the copilot's face and words in a box along the bottom; the words blink.
func paintCopilot(drawing *draw2d.List, words string, now float64) {
	box := draw2d.Rect{X: copilotMargin, Y: ctr.ScreenHeight - copilotMargin - copilotHeight, Width: ctr.ScreenWidth - 2*copilotMargin, Height: copilotHeight}
	drawing.Rectangle(box, 0, hudColours.box, hudColours.edge, 2)
	faceTop := box.Y + (copilotHeight-copilotFaceSize)/2
	for row, line := range copilotFace {
		for column := range len(line) {
			colour := hudColours.face
			switch line[column] {
			case '.':
				continue
			case '+':
				colour = hudColours.eyes
			}
			drawing.Rectangle(draw2d.Rect{X: box.X + 10 + float32(column)*2.4, Y: faceTop + float32(row)*2.4, Width: 2.4, Height: 2.4}, 0, colour, draw2d.Color{}, 0)
		}
	}
	if math.Mod(now*copilotBlinks, 1) < 0.75 {
		drawing.Text(words, box.X+48, box.Y+copilotHeight/2+5, pixelTextSize, draw2d.Regular, hudColours.ink)
	}
}
