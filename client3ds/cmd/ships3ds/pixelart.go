//go:build nintendo3ds

package main

import (
	"math"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
)

// The touch screen's windows, drawn pixel by pixel as handheld games drew theirs: pale windows
// with corners cut in steps and a hard shadow, floating over a blue night with stars, the pages
// as tabs along the top and a little star for a cursor. Sizes are in pixels.
const (
	cornerStep    = 2 // each step of a corner's staircase
	cornerSteps   = 3
	frameWidth    = 3
	shadowDrop    = 4
	tabsTop       = 7
	tabHeight     = 18
	tabPadding    = 6
	tabGap        = 3
	ribbonHeight  = tabsTop + tabHeight // where the tabs end
	rowHeight     = 20
	rowsTop       = ribbonHeight + 8
	textLeft      = 26
	valueLeft     = 196
	hintsHeight   = 18
	cursorWidth   = 7
	cursorHeight  = 7
	pixelTextSize = 16 // VT323
	labelPixels   = 8  // Silkscreen and Press Start 2P, crisp at multiples of 8
)

// pixel is the windows' palette, "a blue moon": windows of night-blue glass with a pale blue
// edge, the stars and the nebula showing through, starlight ink, and a sky blue (gold, the
// accent) for what is chosen.
var pixel = struct {
	screen, window, tab, frame, shadow, ink, soft, gold, goldLight, chosen, glow, danger, good, star draw2d.Color
}{
	screen: draw2d.Hex("#0a1430"), window: draw2d.Hex("#0e1c4290"), tab: draw2d.Hex("#24406e"),
	frame: draw2d.Hex("#9cc8f2"), shadow: draw2d.Hex("#02060fb0"), ink: draw2d.Hex("#eaf4ff"),
	soft: draw2d.Hex("#9fb6d6"), gold: draw2d.Hex("#5fb0ff"), goldLight: draw2d.Hex("#cfe6ff"),
	chosen: draw2d.Hex("#2a4a80"), glow: draw2d.Hex("#9cc8f233"), danger: draw2d.Hex("#ff7088"),
	good: draw2d.Hex("#5fb0ff"), star: draw2d.Hex("#eaf4ff"),
}

// The nebula and the moon behind the menus: soft clouds of colour and a crescent, in pixels.
var nebula = []struct {
	x, y, size float32
	colour     draw2d.Color
}{
	{60, 40, 220, draw2d.Hex("#5b3fa833")}, {250, 150, 260, draw2d.Hex("#2f6fd033")},
	{140, 210, 160, draw2d.Hex("#c0508a22")}, {300, 20, 120, draw2d.Hex("#4fc3f722")},
}

const (
	moonX, moonY, moonSize = 262, 168, 96
	moonShadowShift        = 18
)

var (
	moonLight  = draw2d.Hex("#dfe9f8")
	moonCrater = draw2d.Hex("#c3d2ea")
)

// fullScreen is the touch screen's area.
var fullScreen = draw2d.Rect{Width: ctr.BottomWidth, Height: ctr.BottomHeight}

// stepped fills area with its corners cut in a staircase of steps steps: one rectangle a step,
// the widest in the middle.
func stepped(drawing *draw2d.List, area draw2d.Rect, steps int, colour draw2d.Color) {
	for step := range steps + 1 {
		inset := float32((steps - step) * cornerStep)
		height := float32(step * cornerStep)
		drawing.Rectangle(draw2d.Rect{X: area.X + height, Y: area.Y + inset, Width: area.Width - 2*height, Height: area.Height - 2*inset}, 0, colour, draw2d.Color{}, 0)
	}
}

// inset is area shrunk by a margin on every side.
func inset(area draw2d.Rect, margin float32) draw2d.Rect {
	return draw2d.Rect{X: area.X + margin, Y: area.Y + margin, Width: area.Width - 2*margin, Height: area.Height - 2*margin}
}

// window draws a window over area: a hard shadow, a white edge and the pale inside.
func window(drawing *draw2d.List, area draw2d.Rect) {
	shadow := area
	shadow.X, shadow.Y = shadow.X+shadowDrop, shadow.Y+shadowDrop
	stepped(drawing, shadow, cornerSteps, pixel.shadow)
	stepped(drawing, area, cornerSteps, pixel.frame)
	stepped(drawing, inset(area, frameWidth), cornerSteps, pixel.window)
}

// tab draws one page's tab with its name; the open one in the accent.
func tab(drawing *draw2d.List, area draw2d.Rect, name string, open bool) {
	fill, ink := pixel.tab, pixel.soft
	if open {
		fill, ink = pixel.gold, pixel.screen
	}
	stepped(drawing, area, 1, fill)
	drawing.Text(name, area.X+tabPadding, area.Y+area.Height-5, labelPixels, draw2d.Mono, ink)
}

// The cursor's bobbing: how far, in pixels, and how fast, in turns a second.
const (
	cursorBob      = 2
	cursorBobSpeed = 1.6
)

// cursor draws the little star beside the chosen row, bobbing to and fro.
func cursor(drawing *draw2d.List, x, y float32, now float64) {
	bob := float32(math.Round(cursorBob * math.Sin(now*cursorBobSpeed*2*math.Pi)))
	bitmap(drawing, cursorStar, x+bob, y, pixel.gold, pixel.goldLight)
}

// The light that sweeps along the chosen row: its width in pixels and its passes a second.
const (
	sweepWidth  = 36
	sweepSpeed  = 0.7
	sweepBlocks = 6
)

// shine draws a light sweeping along a band, in blocks fading to its sides.
func shine(drawing *draw2d.List, band draw2d.Rect, now float64) {
	travel := band.Width + sweepWidth
	start := band.X - sweepWidth + float32(math.Mod(now*sweepSpeed, 1))*travel
	block := float32(sweepWidth) / sweepBlocks
	for index := range sweepBlocks {
		left := max(start+float32(index)*block, band.X)
		right := min(start+float32(index+1)*block, band.X+band.Width)
		if right <= left {
			continue
		}
		strength := 1 - abs(float32(index)-(sweepBlocks-1)/2.0)/(sweepBlocks/2.0)
		colour := pixel.glow
		colour[3] *= strength
		drawing.Rectangle(draw2d.Rect{X: left, Y: band.Y, Width: right - left, Height: band.Height}, 0, colour, draw2d.Color{}, 0)
	}
}

// hints draws the buttons' hints along the bottom of area, in the figures' face.
func hints(drawing *draw2d.List, area draw2d.Rect, text string) {
	drawing.Text(text, area.X+12, area.Y+area.Height-8, labelPixels, draw2d.Mono, pixel.soft)
}

// pixelText draws a line of text; value colours it.
func pixelText(drawing *draw2d.List, text string, x, top float32, colour draw2d.Color) {
	drawing.Text(text, x, top+pixelTextSize-3, pixelTextSize, draw2d.Regular, colour)
}

// The starfield behind the menus: how many stars, and how fast the nearest drift, in pixels a
// second; farther layers drift slower and dimmer.
const (
	menuStarCount     = 60
	menuStarLayers    = 3
	menuStarSpeed     = 18
	menuStarTwinkling = 3 // radians a second of a star's twinkle
)

// starfield fills the screen with the night: a nebula, a crescent moon and stars drifting
// left, the near ones faster.
func starfield(drawing *draw2d.List, now float64) {
	drawing.Rectangle(fullScreen, 0, pixel.screen, draw2d.Color{}, 0)
	for _, cloud := range nebula {
		for ring := float32(1); ring >= 0.4; ring -= 0.3 {
			size := cloud.size * ring
			drawing.Rectangle(draw2d.Rect{X: cloud.x - size/2, Y: cloud.y - size/2, Width: size, Height: size}, size/2, cloud.colour, draw2d.Color{}, 0)
		}
	}
	drawing.Rectangle(draw2d.Rect{X: moonX, Y: moonY, Width: moonSize, Height: moonSize}, moonSize/2, moonLight, draw2d.Color{}, 0)
	drawing.Rectangle(draw2d.Rect{X: moonX + 20, Y: moonY + 22, Width: 18, Height: 12}, 6, moonCrater, draw2d.Color{}, 0)
	drawing.Rectangle(draw2d.Rect{X: moonX + 46, Y: moonY + 50, Width: 12, Height: 8}, 4, moonCrater, draw2d.Color{}, 0)
	shade := draw2d.Rect{X: moonX - moonShadowShift, Y: moonY - moonShadowShift/2, Width: moonSize, Height: moonSize}
	drawing.Rectangle(shade, moonSize/2, pixel.screen, draw2d.Color{}, 0) // the dark side cuts a crescent
	for index := range menuStarCount {
		layer := index % menuStarLayers
		depth := float32(menuStarLayers-layer) / menuStarLayers // 1 is the nearest
		seed := uint32(index)*2654435761 + 12345
		baseX := float32(seed%ctr.BottomWidth) - float32(now)*menuStarSpeed*depth
		x := float32(math.Mod(float64(baseX), ctr.BottomWidth))
		if x < 0 {
			x += ctr.BottomWidth
		}
		y := float32((seed / ctr.BottomWidth) % ctr.BottomHeight)
		colour := pixel.star
		colour[3] = depth * float32(0.55+0.45*math.Sin(now*menuStarTwinkling+float64(index)))
		size := float32(1)
		if layer == 0 && index%4 == 0 {
			size = 2
		}
		drawing.Rectangle(draw2d.Rect{X: x, Y: y, Width: size, Height: size}, 0, colour, draw2d.Color{}, 0)
	}
}

// pixelSwitch draws an on / off switch: a round track with the knob to its side, blue when on.
func pixelSwitch(drawing *draw2d.List, x, top float32, on bool) {
	const width, height, knob = 26, 12, 10
	y := top + (rowHeight-2-height)/2
	track, knobX, label := pixel.tab, x+1, "OFF"
	if on {
		track, knobX, label = pixel.good, x+width-knob-1, "ON"
	}
	drawing.Rectangle(draw2d.Rect{X: x, Y: y, Width: width, Height: height}, height/2, track, draw2d.Color{}, 0)
	drawing.Rectangle(draw2d.Rect{X: knobX, Y: y + 1, Width: knob, Height: height - 2}, knob/2, pixel.frame, draw2d.Color{}, 0)
	drawing.Text(label, x+width+8, y+height-2, labelPixels, draw2d.Mono, pixel.soft)
}

// energyBar draws a level as round cells, lit to the level.
func energyBar(drawing *draw2d.List, x, top float32, level, steps int) {
	const width, gap, height = 7, 2, 10
	y := top + (rowHeight-2-height)/2
	for cell := range steps {
		colour := pixel.tab
		if cell < level {
			colour = pixel.gold
		}
		drawing.Rectangle(draw2d.Rect{X: x + float32(cell*(width+gap)), Y: y, Width: width, Height: height}, width/2, colour, draw2d.Color{}, 0)
	}
}
