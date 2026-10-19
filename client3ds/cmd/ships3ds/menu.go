//go:build nintendo3ds

package main

import (
	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
)

// menuRow is one line of a menu page: a choice when it can be activated or adjusted, otherwise a
// line to read.
type menuRow struct {
	label    string
	value    string
	level    int  // drawn as a bar of levelSteps blocks when hasLevel
	hasLevel bool // the value is a level, 0 to levelSteps
	isSwitch bool // the value is a switch, on or off
	on       bool
	colour   draw2d.Color // the label's colour; none is the usual ink
	activate func()       // A, or a touch on the label
	adjust   func(step int)
}

// chooseable tells whether the cursor stops on the row.
func (row menuRow) chooseable() bool { return row.activate != nil || row.adjust != nil }

// menuPage is a window of rows with a title; L and R go through the pages of a menu.
type menuPage struct {
	title      string
	rows       []menuRow
	valueLeft  float32 // where values start, in pixels; 0 is valueLeft
	newestLast bool    // a log: it opens scrolled to its end
	small      bool    // a window in the middle of the screen rather than over all of it
}

// levelSteps is how many blocks a level's bar has; a volume of 0.7 shows seven.
const levelSteps = 10

// The menu's measures, in pixels.
const (
	visibleRows       = (ctr.BottomHeight - 2*fullPageMargin - rowsTop - hintsHeight) / rowHeight
	smallWindowWidth  = 280
	smallWindowMargin = 12
	fullPageMargin    = 3  // the stars show around a page
	labelCharacters   = 36 // how much of a line fits across the window, in VT323
)

// menu is where the cursor is in the pages shown; it lasts while the same pages are shown.
type menu struct {
	page, cursor, scroll int
	opened               string  // the first page's title when the menu was set up: other pages, other menu
	openedAt             float64 // when, in seconds: the window unfolds from then
}

// unfoldSeconds is how long a window takes to unfold when it opens.
const unfoldSeconds = 0.18

// prepare puts the cursor on the page's first choice when the pages are new, or a page changed.
func (shown *menu) prepare(pages []menuPage, now float64) {
	if len(pages) == 0 {
		return
	}
	if pages[0].title != shown.opened {
		shown.opened, shown.page, shown.openedAt = pages[0].title, 0, now
		shown.turnTo(pages[0])
	}
	shown.page = min(shown.page, len(pages)-1)
	page := pages[shown.page]
	if shown.cursor < 0 || shown.cursor >= len(page.rows) || !page.rows[shown.cursor].chooseable() {
		shown.cursor = firstChoice(page)
	}
}

// turnTo puts the cursor at the top of a page just turned to, or a log's end.
func (shown *menu) turnTo(page menuPage) {
	shown.cursor, shown.scroll = firstChoice(page), 0
	if page.newestLast {
		shown.scroll = max(len(page.rows)-visibleRows, 0)
	}
}

// firstChoice is the index of the page's first chooseable row, or -1 when it has none.
func firstChoice(page menuPage) int {
	for index, row := range page.rows {
		if row.chooseable() {
			return index
		}
	}
	return -1
}

// move goes to the next choice up or down (step -1 or 1), keeping it in sight; on a page with
// nothing to choose it scrolls instead. It tells whether anything moved.
func (shown *menu) move(page menuPage, step int) bool {
	if shown.cursor < 0 {
		scroll := min(max(shown.scroll+step, 0), max(len(page.rows)-visibleRows, 0))
		moved := scroll != shown.scroll
		shown.scroll = scroll
		return moved
	}
	for index := shown.cursor + step; index >= 0 && index < len(page.rows); index += step {
		if page.rows[index].chooseable() {
			shown.cursor = index
			shown.keepInSight(page)
			return true
		}
	}
	return false
}

// keepInSight scrolls so that the cursor's row shows, and the lines just above it too.
func (shown *menu) keepInSight(page menuPage) {
	top := shown.cursor
	for top > 0 && !page.rows[top-1].chooseable() && shown.cursor-top < visibleRows-1 {
		top-- // a heading above a choice comes into sight with it
	}
	if top < shown.scroll {
		shown.scroll = top
	}
	if shown.cursor >= shown.scroll+visibleRows {
		shown.scroll = shown.cursor - visibleRows + 1
	}
}

// menuInput is what the menu reads of a frame: the buttons pressed and where the screen was
// touched.
type menuInput struct {
	pressed            ctr.Buttons
	touchX, touchY     float32
	closes, turnsPages bool // B closes the menu; L and R turn its pages
}

// press acts on a frame's buttons and touch over pages, with the menu's sounds: the cross moves,
// A activates, left and right adjust, L and R turn pages. It tells whether B asked to close.
func (session *session) pressMenu(pages []menuPage, input menuInput) (closed bool) {
	shown := &session.menu
	shown.prepare(pages, session.now)
	if len(pages) == 0 {
		return false
	}
	page := pages[shown.page]
	pressed := input.pressed
	switch {
	case pressed&ctr.ButtonB != 0 && input.closes:
		session.sounds.play(session.sounds.back, 1)
		return true
	case pressed&(ctr.ButtonL|ctr.ButtonR) != 0 && input.turnsPages && len(pages) > 1:
		step := 1
		if pressed&ctr.ButtonL != 0 {
			step = -1
		}
		session.turnPage(pages, step)
	case pressed&ctr.ButtonUp != 0:
		session.moved(shown.move(page, -1))
	case pressed&ctr.ButtonDown != 0:
		session.moved(shown.move(page, 1))
	case pressed&ctr.ButtonTouch != 0:
		session.touchMenu(pages, input)
	case shown.cursor < 0:
		return false
	case pressed&ctr.ButtonA != 0:
		session.activate(page.rows[shown.cursor])
	case pressed&ctr.ButtonLeft != 0:
		session.adjust(page.rows[shown.cursor], -1)
	case pressed&ctr.ButtonRight != 0:
		session.adjust(page.rows[shown.cursor], 1)
	}
	return false
}

// turnPage goes step pages on, round the ends.
func (session *session) turnPage(pages []menuPage, step int) {
	shown := &session.menu
	shown.page = (shown.page + step + len(pages)) % len(pages)
	shown.turnTo(pages[shown.page])
	session.sounds.play(session.sounds.move, 1)
}

func (session *session) moved(moved bool) {
	if moved {
		session.sounds.play(session.sounds.move, 1)
	}
}

func (session *session) activate(row menuRow) {
	if row.activate == nil {
		session.adjust(row, 1) // a toggle: A flips it as the cross does
		return
	}
	session.sounds.play(session.sounds.choose, 1)
	row.activate()
}

func (session *session) adjust(row menuRow, step int) {
	if row.adjust == nil {
		return
	}
	session.sounds.play(session.sounds.move, 1)
	row.adjust(step)
}

// touchMenu acts on a touch: the ribbon's ends turn the page, a row is chosen and activated, a
// touch on a value's left half lowers it and on its right half raises it.
func (session *session) touchMenu(pages []menuPage, input menuInput) {
	shown := &session.menu
	page := pages[shown.page]
	area := pageArea(page)
	if input.touchY < area.Y+ribbonHeight {
		for index, place := range tabPlaces(area, pages) {
			if input.turnsPages && index != shown.page && input.touchX >= place.X && input.touchX < place.X+place.Width {
				session.turnPage(pages, index-shown.page)
			}
		}
		return
	}
	index := shown.scroll + int((input.touchY-area.Y-rowsTop)/rowHeight)
	if input.touchY < area.Y+rowsTop || index >= len(page.rows) || index >= shown.scroll+visibleRows || !page.rows[index].chooseable() {
		return
	}
	shown.cursor = index
	row := page.rows[index]
	valueStart := area.X + valueLeftOf(page)
	if row.adjust != nil && input.touchX >= valueStart {
		step := 1
		if input.touchX < valueStart+(area.X+area.Width-valueStart)/2 {
			step = -1
		}
		session.adjust(row, step)
		return
	}
	session.activate(row)
}

// pageArea is where a page's window is: the whole screen, or a small window in its middle.
func pageArea(page menuPage) draw2d.Rect {
	if !page.small {
		return inset(fullScreen, fullPageMargin)
	}
	height := float32(rowsTop + len(page.rows)*rowHeight + hintsHeight + smallWindowMargin)
	return draw2d.Rect{X: (ctr.BottomWidth - smallWindowWidth) / 2, Y: (ctr.BottomHeight - height) / 2, Width: smallWindowWidth, Height: height}
}

func valueLeftOf(page menuPage) float32 {
	if page.valueLeft > 0 {
		return page.valueLeft
	}
	return valueLeft
}

// paintPages draws the page the menu is on, with its rows, cursor, page marks and hints.
func (session *session) paintPages(drawing *draw2d.List, pages []menuPage, hint string) {
	shown := &session.menu
	shown.prepare(pages, session.now)
	if len(pages) == 0 {
		return
	}
	page := pages[shown.page]
	area := pageArea(page)
	starfield(drawing, session.now)
	if unfolded := (session.now - shown.openedAt) / unfoldSeconds; unfolded < 1 {
		// The window unfolds from its middle line, as the old handhelds opened theirs.
		height := max(float32(unfolded)*area.Height, 8)
		window(drawing, draw2d.Rect{X: area.X, Y: area.Y + (area.Height-height)/2, Width: area.Width, Height: height})
		return
	}
	window(drawing, area)
	for index, place := range tabPlaces(area, pages) {
		tab(drawing, place, pages[index].title, index == shown.page)
	}
	for line := range min(visibleRows, len(page.rows)-shown.scroll) {
		index := shown.scroll + line
		paintRow(drawing, area, page, page.rows[index], area.Y+rowsTop+float32(line*rowHeight), index == shown.cursor, session.now)
	}
	if shown.scroll > 0 {
		paintScrollMark(drawing, area, area.Y+rowsTop-6, -1)
	}
	if shown.scroll+visibleRows < len(page.rows) {
		paintScrollMark(drawing, area, area.Y+area.Height-hintsHeight-4, 1)
	}
	hints(drawing, area, hint)
}

// paintRow draws one row; the chosen one on a band, with the cursor before it.
func paintRow(drawing *draw2d.List, area draw2d.Rect, page menuPage, row menuRow, top float32, chosen bool, now float64) {
	if chosen {
		band := draw2d.Rect{X: area.X + frameWidth + 6, Y: top, Width: area.Width - 2*frameWidth - 12, Height: rowHeight - 2}
		stepped(drawing, band, 1, pixel.chosen)
		shine(drawing, band, now)
		cursor(drawing, area.X+12, top+(rowHeight-2-cursorHeight)/2, now)
	}
	ink := pixel.soft
	if row.chooseable() {
		ink = pixel.ink
	}
	if row.colour != (draw2d.Color{}) {
		ink = row.colour
	}
	pixelText(drawing, row.label, area.X+textLeft, top, ink)
	valueX := area.X + valueLeftOf(page)
	switch {
	case row.hasLevel:
		energyBar(drawing, valueX, top, row.level, levelSteps)
		return
	case row.isSwitch:
		pixelSwitch(drawing, valueX, top, row.on)
		return
	}
	valueInk := pixel.soft
	if chosen && row.adjust != nil {
		valueInk = pixel.gold
	}
	pixelText(drawing, row.value, valueX, top, valueInk)
}

// tabPlaces are where the pages' tabs go along the window's top, one after another: the
// figures' face is 8 pixels a letter, so no measuring is needed.
func tabPlaces(area draw2d.Rect, pages []menuPage) []draw2d.Rect {
	places := make([]draw2d.Rect, len(pages))
	x := area.X + 8
	for index, page := range pages {
		width := float32(len(page.title)*labelPixels + 2*tabPadding)
		places[index] = draw2d.Rect{X: x, Y: area.Y + tabsTop, Width: width, Height: tabHeight}
		x += width + tabGap
	}
	return places
}

// paintScrollMark draws a small triangle pointing up (-1) or down (1): more rows that way.
func paintScrollMark(drawing *draw2d.List, area draw2d.Rect, y float32, direction int) {
	const halfWidth = 4
	x := area.X + area.Width - 18
	for row := range halfWidth {
		width := float32(2 * (row + 1))
		rowY := y + float32(row)
		if direction > 0 {
			rowY = y + float32(halfWidth-1-row)
		}
		drawing.Rectangle(draw2d.Rect{X: x - width/2, Y: rowY, Width: width, Height: 1}, 0, pixel.gold, draw2d.Color{}, 0)
	}
}

// fitted shortens a line to what fits across a window, with an ellipsis.
func fitted(text string, characters int) string {
	runes := []rune(text)
	if len(runes) <= characters {
		return text
	}
	return string(runes[:characters-1]) + "…"
}

// wrapped breaks text into lines of at most characters, between words where it can.
func wrapped(text string, characters int) []string {
	lines := []string{}
	runes := []rune(text)
	for len(runes) > characters {
		cut := characters
		for space := characters; space > characters/2; space-- {
			if runes[space] == ' ' {
				cut = space
				break
			}
		}
		lines = append(lines, string(runes[:cut]))
		runes = runes[cut:]
		for len(runes) > 0 && runes[0] == ' ' {
			runes = runes[1:]
		}
	}
	return append(lines, string(runes))
}
