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
	lifeSegmentWidth  = 10
	lifeSegmentHeight = 10
	lifeSegmentGap    = 2
	lifeHeartRoom     = 24 // the heart's place at the bar's left
	lifeBeatsASecond  = 1.6
	copilotPixel      = 2.4
	lifeLowShare      = 0.3 // below this much life the charges turn red
	copilotHeight     = 44
	copilotMargin     = 8
	copilotWarnShare  = 0.35 // how near the edge the copilot speaks up
	copilotBlinks     = 2.0  // the warning's blinks a second
)

// copilotEdgeWarning is what the copilot says as the ship heads out of the arena.
const copilotEdgeWarning = "We're leaving the combat arena, sir!"

// copilotTurningBack is what the copilot says while it turns the ship back itself.
const copilotTurningBack = "Taking the helm, sir. Back to the arena!"

// copilotFace is the copilot, a little round robot: a shell ('#', '='), a dark screen for a
// face ('-'), big eyes with a shine ('o', '*'), blushing cheeks ('p'), a smile ('s') and an
// antenna with a light on top ('a').
var copilotFace = []string{
	"......aa......",
	"......aa......",
	".......#......",
	"...########...",
	"..##------##..",
	".##--------##.",
	".#--oo--oo--#.",
	".#-o*o--o*o-#.",
	".#--oo--oo--#.",
	".#pp------pp#.",
	".#----ss----#.",
	".##--------##.",
	"..##======##..",
	"...########...",
}

// copilotPalette are the copilot's colours, by letter.
var copilotPalette = map[byte]draw2d.Color{
	'#': draw2d.Hex("#dfe9f8"), '=': draw2d.Hex("#9cb4d6"), '-': draw2d.Hex("#16233f"),
	'o': draw2d.Hex("#5ee0ff"), '*': draw2d.Hex("#ffffff"), 'p': draw2d.Hex("#ff8fb8"),
	's': draw2d.Hex("#5ee0ff"), 'a': draw2d.Hex("#ff8fb8"),
}

// lifeHeart is the bar's heart, and its colours.
var (
	lifeHeart = []string{
		".##.##.",
		"#*#####",
		"#######",
		".#####.",
		"..###..",
		"...#...",
	}
	heartPalette = map[byte]draw2d.Color{'#': draw2d.Hex("#ff6f91"), '*': draw2d.Hex("#ffd6e0")}
)

// The life bar's cells run from a light blue to a deep one; low, they are red.
var (
	lifeCells    = [2]draw2d.Color{draw2d.Hex("#7fe8ff"), draw2d.Hex("#3d8fe0")}
	lifeCellsLow = [2]draw2d.Color{draw2d.Hex("#ff9aa8"), draw2d.Hex("#e0405a")}
)

// hudColours are the readings' colours: the spent charges, the boxes and their shine.
var hudColours = struct {
	spent, box, edge, ink, shadow, shine draw2d.Color
}{
	spent: draw2d.Hex("#1c2a48"), box: draw2d.Hex("#0e1c42e0"), edge: draw2d.Hex("#9cc8f2"),
	ink: draw2d.Hex("#eaf4ff"), shadow: draw2d.Hex("#02060fa0"), shine: draw2d.Hex("#ffffff70"),
}

// paintHUD lays the readings out in drawing for this frame, which it empties first.
func paintHUD(drawing *draw2d.List, state *game.Game, now float64) {
	drawing.Reset()
	paintLife(drawing, state.Player.Life, state.Settings.MaxLife, now)
	switch {
	case state.Returning:
		paintCopilot(drawing, copilotTurningBack, now)
	case state.EdgeNearness() >= copilotWarnShare:
		paintCopilot(drawing, copilotEdgeWarning, now)
	}
}

// paintLife draws life as charges in a capsule with a heart: one cell for each tenth of the
// most life, lit as far as life reaches, the last one partly, each with a shine along its top.
// Low, the cells turn red and the heart beats.
func paintLife(drawing *draw2d.List, life, most, now float64) {
	if most <= 0 {
		return
	}
	segments := min(int(math.Ceil(most)), lifeMostSegments)
	share := life / most
	cells := lifeCells
	low := share < lifeLowShare
	if low {
		cells = lifeCellsLow
	}
	width := float32(lifeHeartRoom + segments*(lifeSegmentWidth+lifeSegmentGap) + lifeSegmentGap + 2)
	frame := draw2d.Rect{X: lifeLeft, Y: lifeTop, Width: width, Height: lifeSegmentHeight + 8}
	shadow := frame
	shadow.X, shadow.Y = shadow.X+2, shadow.Y+2
	stepped(drawing, shadow, 2, hudColours.shadow)
	stepped(drawing, frame, 2, hudColours.edge)
	stepped(drawing, inset(frame, 2), 2, hudColours.box)
	heartSize := float32(2)
	if low && math.Mod(now*lifeBeatsASecond, 1) < 0.3 {
		heartSize = 2.4
	}
	paintedBitmap(drawing, lifeHeart, frame.X+7, frame.Y+(frame.Height-6*heartSize)/2, heartSize, heartPalette)
	left := frame.X + lifeHeartRoom
	top := frame.Y + 4
	for segment := range segments {
		x := left + float32(segment*(lifeSegmentWidth+lifeSegmentGap))
		drawing.Rectangle(draw2d.Rect{X: x, Y: top, Width: lifeSegmentWidth, Height: lifeSegmentHeight}, 0, hudColours.spent, draw2d.Color{}, 0)
		filled := min(max(share*float64(segments)-float64(segment), 0), 1)
		if filled == 0 {
			continue
		}
		colour := mixColours(cells[0], cells[1], float32(segment)/float32(max(segments-1, 1)))
		fill := float32(filled) * lifeSegmentWidth
		drawing.Rectangle(draw2d.Rect{X: x, Y: top, Width: fill, Height: lifeSegmentHeight}, 0, colour, draw2d.Color{}, 0)
		drawing.Rectangle(draw2d.Rect{X: x, Y: top, Width: fill, Height: 2}, 0, hudColours.shine, draw2d.Color{}, 0)
	}
}

// mixColours is between first and second, share of the way.
func mixColours(first, second draw2d.Color, share float32) draw2d.Color {
	var mixed draw2d.Color
	for index := range mixed {
		mixed[index] = first[index] + (second[index]-first[index])*share
	}
	return mixed
}

// paintCopilot draws the copilot's face and words in a box along the bottom; the words blink.
func paintCopilot(drawing *draw2d.List, words string, now float64) {
	box := draw2d.Rect{X: copilotMargin, Y: ctr.ScreenHeight - copilotMargin - copilotHeight, Width: ctr.ScreenWidth - 2*copilotMargin, Height: copilotHeight}
	shadow := box
	shadow.X, shadow.Y = shadow.X+2, shadow.Y+2
	stepped(drawing, shadow, 2, hudColours.shadow)
	stepped(drawing, box, 2, hudColours.edge)
	stepped(drawing, inset(box, 2), 2, hudColours.box)
	faceTop := box.Y + (copilotHeight-float32(len(copilotFace))*copilotPixel)/2
	paintedBitmap(drawing, copilotFace, box.X+8, faceTop, copilotPixel, copilotPalette)
	if math.Mod(now*copilotBlinks, 1) < 0.75 {
		drawing.Text(words, box.X+52, box.Y+copilotHeight/2+5, pixelTextSize, draw2d.Regular, hudColours.ink)
	}
}
