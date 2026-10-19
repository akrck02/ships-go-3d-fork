//go:build nintendo3ds

package main

import "github.com/akrck02/go-renderer/draw2d"

// pageIcons are the menu's pages' icons on their tabs, 12 × 12 pixels: '#' the tab's ink, '+'
// lighter.
var pageIcons = map[string][]string{
	pilotPage: {
		"....####....",
		"..########..",
		".##++++++##.",
		".#+######+#.",
		"##+#....#+##",
		"##+#....#+##",
		"##+######+##",
		".##++++++##.",
		"..########..",
		"...#....#...",
		"..##....##..",
		"............",
	},
	figuresPage: {
		"............",
		"..........##",
		"..........##",
		"......##..##",
		"......##..##",
		"..##..##..##",
		"..##..##..##",
		"..##..##..##",
		"++++++++++++",
		"............",
		"............",
		"............",
	},
	settingsPage: {
		".....##.....",
		"..#.####.#..",
		".##########.",
		"..###++###..",
		".###+..+###.",
		"####+..+####",
		".###+..+###.",
		"..###++###..",
		".##########.",
		"..#.####.#..",
		".....##.....",
		"............",
	},
	accountPage: {
		"....####....",
		"...######...",
		"...##++##...",
		"...######...",
		"....####....",
		"............",
		"..########..",
		".##########.",
		".###++++###.",
		".##########.",
		"............",
		"............",
	},
	controlsPage: {
		"............",
		"............",
		"..########..",
		".##########.",
		"##+###++#+##",
		"#+++###+#+##",
		"##+####+++##",
		"############",
		"###......###",
		".##......##.",
		"............",
		"............",
	},
}

// cursorStar is the cursor: a little four-pointed star, '#' in the accent and '+' lighter.
var cursorStar = []string{
	"...#...",
	"...#...",
	"..#+#..",
	"##+++##",
	"..#+#..",
	"...#...",
	"...#...",
}

// The titles of the windows that are not the menu's pages.
const (
	connectingPage = "CONNECTING"
	restartPage    = "RESTART?"
	joiningPage    = "JOINING"
)

// bitmap draws a picture drawn by hand in text with its top left at x, y: '#' in main, '+' in light.
func bitmap(drawing *draw2d.List, rows []string, x, y float32, main, light draw2d.Color) {
	for row, line := range rows {
		for column := 0; column < len(line); column++ {
			colour := main
			switch line[column] {
			case '.':
				continue
			case '+':
				colour = light
			}
			// Runs of one colour in one rectangle: fewer to draw.
			end := column + 1
			for end < len(line) && line[end] == line[column] {
				end++
			}
			drawing.Rectangle(draw2d.Rect{X: x + float32(column), Y: y + float32(row), Width: float32(end - column), Height: 1}, 0, colour, draw2d.Color{}, 0)
			column = end - 1
		}
	}
}
