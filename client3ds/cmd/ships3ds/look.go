//go:build nintendo3ds

package main

import (
	_ "embed"

	"github.com/akrck02/go-renderer/draw2d"
	"github.com/akrck02/go-renderer/opengl/overlay"
	ui "github.com/akrck02/go-ui"

	"github.com/akrck02/go-3ds/bottom"
)

// The game's look on the console: pixel fonts and square corners, as handheld space shooters
// had, in the colours of space. The fonts are under the SIL Open Font License (fonts/OFL-*.txt),
// from Google Fonts: VT323 for text, Silkscreen for headings and names, Press Start 2P for
// figures. Each is crisp at its own sizes only: VT323 at 16, the others at multiples of 8.
var (
	//go:embed fonts/VT323-Regular.ttf
	vt323 []byte
	//go:embed fonts/Silkscreen-Bold.ttf
	silkscreenBold []byte
	//go:embed fonts/PressStart2P-Regular.ttf
	pressStart []byte
)

// The pixel fonts' sizes, in pixels.
const (
	textSize    = 16 // VT323: text, captions
	headingSize = 16 // Silkscreen: headings, buttons' labels in bold
	titleSize   = 40 // Silkscreen: the game's name
)

// usePixelFonts makes every text draw in the pixel fonts; call it before the first painter is
// made, which reads them.
func usePixelFonts() {
	overlay.UseFonts(overlay.Fonts{Regular: vt323, Medium: vt323, Bold: silkscreenBold, Mono: pressStart})
}

// spaceTheme is go-3ds's bottom-screen theme in pixel art and the colours of space: a night blue,
// pale ink, and an arcade gold for what is chosen; square corners and hard shadows.
func spaceTheme() ui.Theme {
	theme := bottom.Theme()
	theme.Background = draw2d.Hex("#070b1a")
	theme.Surface = draw2d.Hex("#0e1530")
	theme.Sunken = draw2d.Hex("#18214a")
	theme.Line = draw2d.Hex("#2b3a78")
	theme.ControlLine = draw2d.Hex("#4458b0")
	theme.Ink = draw2d.Hex("#e9edff")
	theme.InkSoft = draw2d.Hex("#aab4ea")
	theme.InkQuiet = draw2d.Hex("#7a86c4")
	theme.Primary = draw2d.Hex("#ffcc33")
	theme.OnPrimary = draw2d.Hex("#1c1400")
	theme.PrimaryTint = draw2d.Hex("#3a3112")
	theme.FocusRing = draw2d.Hex("#ffcc3380")
	theme.Shadow = draw2d.Hex("#000000")
	theme.TextSize = textSize
	theme.SmallTextSize = textSize
	theme.HeadingSize = headingSize
	theme.IconSize = 16
	theme.ControlRadius = 0
	theme.PanelRadius = 0
	theme.ShadowBlur = 0 // a hard shadow, a pixel's step below
	theme.ShadowDrop = 2
	return theme
}
