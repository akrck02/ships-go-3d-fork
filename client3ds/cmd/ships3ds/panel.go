//go:build nintendo3ds

package main

import (
	"math"
	"sort"
	"strconv"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"

	"ships3d/client3ds/profile"
)

// The menu's pages, by title.
const (
	pilotPage    = "PILOT"
	figuresPage  = "STATS"
	settingsPage = "SETUP"
	accountPage  = "ACCOUNT"
	controlsPage = "CONTROLS"
	logPage      = "LOG"
)

// figuresPageIndex is where the figures are in the game's menu, which reads the heap's size
// while they show.
const figuresPageIndex = 1

// The buttons' hints along the bottom of the windows.
const (
	menuHint    = "A OK  B BACK  L/R PAGE"
	choiceHint  = "A OK  B BACK"
	waitingHint = "PLEASE WAIT"
)

// build lays out the touch screen for what it shows now: all of it painted pixel by pixel.
func (session *session) build(context *ui.Context) {
	layout := ui.Layout{Width: ui.Fixed(ctr.BottomWidth), Height: ui.Fixed(ctr.BottomHeight)}
	switch {
	case session.screen == titleScreen && session.confirming == noChange:
		context.Paint(layout, session.paintTitlePrompt)
	case session.screen == shipScreen && !session.menuOpen && session.confirming == noChange:
		context.Paint(layout, session.paintShipChoice)
	case session.screen == gameScreen && session.game.Welcomed && !session.menuOpen:
		session.buildMap(context)
	default:
		context.Paint(layout, func(area draw2d.Rect, drawing *draw2d.List) {
			pages, hint := session.pages()
			session.paintPages(drawing, pages, hint)
		})
	}
}

// pages are the menu's pages shown now, and the hint for its buttons.
func (session *session) pages() ([]menuPage, string) {
	switch {
	case session.confirming != noChange:
		return []menuPage{session.restartWarning()}, choiceHint
	case session.screen == enteringScreen:
		page := session.enteringPage()
		if session.asking == askEnter {
			return []menuPage{page}, waitingHint
		}
		return []menuPage{page}, "A OK"
	case session.screen == shipScreen:
		return []menuPage{session.accountPage(), session.settingsPage(), controlsPageOf(), session.logPage()}, menuHint
	case session.menuOpen:
		return []menuPage{session.pilotPage(), session.figuresPage(), session.settingsPage(), session.accountPage(), session.logPage()}, menuHint
	}
	return []menuPage{session.joiningPage()}, "B SHIPS"
}

// pressPages acts on the menu shown now; B closes it, or answers no.
func (session *session) pressPages(input menuInput) {
	pages, _ := session.pages()
	input.closes = session.confirming != noChange || session.menuOpen || session.screen == gameScreen
	input.turnsPages = session.menuOpen && session.confirming == noChange
	if !session.pressMenu(pages, input) {
		return
	}
	switch {
	case session.confirming != noChange:
		session.confirming = noChange
	case session.menuOpen:
		session.menuOpen = false
	case session.screen == gameScreen:
		session.leave("")
	}
}

// statusRows are the connection's status and the server's complaint, wrapped to the window.
func (session *session) statusRows() []menuRow {
	rows := []menuRow{}
	for _, text := range []string{session.status, session.problem()} {
		if text == "" {
			continue
		}
		for _, line := range wrapped(text, labelCharacters) {
			rows = append(rows, menuRow{label: line, colour: pixel.soft})
		}
	}
	return rows
}

func (session *session) problem() string {
	if session.game.Problem == "" {
		return ""
	}
	return "Server: " + session.game.Problem
}

// restartWarning asks before a change of server or account, which starts the game again.
func (session *session) restartWarning() menuPage {
	return menuPage{title: restartPage, small: true, rows: []menuRow{
		{label: "You will leave the game"},
		{label: "and connect again."},
		{},
		{label: "Restart", colour: pixel.danger, activate: session.confirm},
		{label: "Cancel", activate: func() { session.confirming = noChange }},
	}}
}

// enteringPage says the server is being reached, or why it couldn't be, with a way out.
func (session *session) enteringPage() menuPage {
	rows := session.statusRows()
	if session.asking != askEnter {
		rows = append(rows, menuRow{},
			menuRow{label: "Try again", activate: session.enter},
			menuRow{label: "Server and account", activate: func() {
				session.menuOpen, session.screen = true, shipScreen
				session.menu.opened = "" // the menu opens new, on its first page: the account
			}})
	}
	return menuPage{title: connectingPage, small: true, rows: rows}
}

// joiningPage shows while the game's connection is made, before the server's welcome.
func (session *session) joiningPage() menuPage {
	return menuPage{title: joiningPage, small: true, rows: append(session.statusRows(), menuRow{},
		menuRow{label: "Back to the ships", activate: func() { session.leave("") }})}
}

// pilotPage is the game's own page: life and score, the way back, and the others' scores.
func (session *session) pilotPage() menuPage {
	state := session.game
	rows := []menuRow{
		{label: "Life", value: strconv.Itoa(int(state.Player.Life)) + " / " + strconv.Itoa(int(state.Settings.MaxLife))},
		{label: "Kills / deaths", value: strconv.Itoa(state.Player.Kills) + " / " + strconv.Itoa(state.Player.Deaths)},
		{label: "Resume", activate: func() { session.menuOpen = false }},
		{label: "Choose another ship", activate: func() { session.leave("") }},
		{label: "Quit the game", colour: pixel.danger, activate: func() { session.quitting = true }},
	}
	if len(state.Remotes) > 0 {
		rows = append(rows, menuRow{label: "OTHER PILOTS", colour: pixel.gold})
	}
	return menuPage{title: pilotPage, rows: append(rows, session.remoteRows()...)}
}

// remoteRows are the other players' scores, best first.
func (session *session) remoteRows() []menuRow {
	type score struct {
		name          string
		kills, deaths int
	}
	scores := []score{}
	for _, remote := range session.game.Remotes {
		scores = append(scores, score{remote.Player.Name, remote.Player.Kills, remote.Player.Deaths})
	}
	sort.Slice(scores, func(first, second int) bool { return scores[first].kills > scores[second].kills })
	rows := make([]menuRow, 0, len(scores))
	for _, entry := range scores {
		rows = append(rows, menuRow{label: fitted(entry.name, 20), value: strconv.Itoa(entry.kills) + " / " + strconv.Itoa(entry.deaths)})
	}
	return rows
}

// settingsPage offers how the ship flies, the radar shows and the game sounds; each change is
// remembered.
func (session *session) settingsPage() menuPage {
	settings := session.profile.Settings
	change := func(apply func(settings *profile.Settings)) func(int) {
		return func(step int) {
			changed := session.profile.Settings
			apply(&changed)
			session.remember(func(remembered *profile.Profile) { remembered.Settings = changed })
			session.applyVolumes()
		}
	}
	stepped := func(value *float32, step int) {
		*value = float32(min(max(math.Round(float64(*value*levelSteps))+float64(step), 0), levelSteps)) / levelSteps
	}
	level := func(value float32) int { return int(value*levelSteps + 0.5) }
	return menuPage{title: settingsPage, rows: []menuRow{
		{label: "Up raises the nose", isSwitch: true, on: settings.InvertPitch, adjust: change(func(settings *profile.Settings) { settings.InvertPitch = !settings.InvertPitch })},
		{label: "Radar", value: radarName(settings.FlatRadar), adjust: change(func(settings *profile.Settings) { settings.FlatRadar = !settings.FlatRadar })},
		{label: "Steer by tilting", isSwitch: true, on: settings.Gyroscope, adjust: change(func(settings *profile.Settings) {
			settings.Gyroscope = !settings.Gyroscope
			session.tilt.Centre() // from where the console is held now
		})},
		{label: "Fine aim", hasLevel: true, level: level(settings.Precision), adjust: func(step int) {
			change(func(settings *profile.Settings) { stepped(&settings.Precision, step) })(step)
		}},
		{label: "Effects", hasLevel: true, level: level(settings.Effects), adjust: func(step int) {
			change(func(settings *profile.Settings) { stepped(&settings.Effects, step) })(step)
		}},
		{label: "Music", hasLevel: true, level: level(settings.Music), adjust: func(step int) {
			change(func(settings *profile.Settings) { stepped(&settings.Music, step) })(step)
		}},
		{},
		{label: "Fine aim: the circle pad softer"},
		{label: "near its centre. L centres the tilt."},
	}, valueLeft: 186}
}

func radarName(flat bool) string {
	if flat {
		return "FLAT"
	}
	return "3D"
}

// accountPage shows the server and the account, each with its change, which restarts the game.
func (session *session) accountPage() menuPage {
	rows := []menuRow{
		{label: "SERVER", colour: pixel.gold},
		{label: fitted(session.profile.Server, labelCharacters-2)},
		{label: "Change server", activate: func() { session.ask(changeServer) }},
		{label: "ACCOUNT", colour: pixel.gold},
	}
	if session.account.Token == "" {
		rows = append(rows, menuRow{label: "Guest"}, menuRow{label: "Log in", activate: func() { session.ask(changeLogin) }})
	} else {
		rows = append(rows, menuRow{label: fitted(session.account.Username, labelCharacters-2)},
			menuRow{label: "Log out", activate: func() { session.ask(changeLogout) }})
	}
	return menuPage{title: accountPage, rows: append(rows, session.statusRows()...)}
}

// controlsPageOf lists the controls in two columns.
func controlsPageOf() menuPage {
	rows := make([]menuRow, 0, len(controlLines))
	for _, line := range controlLines {
		rows = append(rows, menuRow{label: line[0], value: line[1], colour: pixel.gold})
	}
	return menuPage{title: controlsPage, rows: rows, valueLeft: controlNameWidth + textLeft}
}

// logPage shows the log's lines, newest last; the whole log is on the SD card, log.txt.
func (session *session) logPage() menuPage {
	rows := []menuRow{}
	for _, line := range session.log.lines {
		for _, part := range wrapped(line, labelCharacters) {
			rows = append(rows, menuRow{label: part, colour: pixel.soft})
		}
	}
	return menuPage{title: logPage, rows: rows, newestLast: true}
}

// The title screen's prompt: it blinks, as arcade screens did, at blinksPerSecond.
const (
	blinksPerSecond = 1.5
	promptY         = 150
)

// paintTitlePrompt draws the title screen's touch screen: a framed window and a blinking prompt;
// the name is on the top screen.
func (session *session) paintTitlePrompt(area draw2d.Rect, drawing *draw2d.List) {
	starfield(drawing, session.now)
	window(drawing, inset(fullScreen, 8))
	session.centredText(drawing, "SHIPS 3D", 80, labelPixels*3, draw2d.Bold, pixel.gold)
	session.centredText(drawing, "a space dogfight", 104, pixelTextSize, draw2d.Regular, pixel.soft)
	if math.Mod(session.now*blinksPerSecond, 1) < 0.65 {
		session.centredText(drawing, "TOUCH OR PRESS A", promptY, labelPixels, draw2d.Mono, pixel.ink)
	}
	hints(drawing, inset(fullScreen, 8), "START QUIT")
}

// The ship screen's measures, in pixels.
const (
	shipNameY      = 34
	shipArrowWidth = 8
	shipArrowGap   = 14
	shipNameWidest = ctr.BottomWidth - 2*(shipArrowGap+shipArrowWidth+10)
)

// paintShipChoice draws over the turning ship: its name between arrows that the cross turns,
// which ship of how many, and the buttons' hints.
func (session *session) paintShipChoice(area draw2d.Rect, drawing *draw2d.List) {
	window(drawing, draw2d.Rect{X: 4, Y: 2, Width: ctr.BottomWidth - 8, Height: shipRowHeight})
	label := shipLabel(session.ships[session.ship])
	size := float32(bigSize)
	width := session.textWidth(label, size, draw2d.Bold)
	if width > shipNameWidest {
		size = headingSize // a long name in the smaller size, so that the arrows stay on the screen
		width = session.textWidth(label, size, draw2d.Bold)
	}
	left := (ctr.BottomWidth - width) / 2
	drawing.Text(label, left, shipNameY, size, draw2d.Bold, pixel.ink)
	arrow(drawing, left-shipArrowGap-shipArrowWidth, shipNameY-14, -1)
	arrow(drawing, left+width+shipArrowGap, shipNameY-14, 1)
	count := strconv.Itoa(session.ship+1) + " / " + strconv.Itoa(len(session.ships)) + "  " + fitted(session.playerName(), 20)
	session.centredText(drawing, count, shipNameY+16, pixelTextSize, draw2d.Regular, pixel.soft)
	bar := draw2d.Rect{X: 4, Y: ctr.BottomHeight - 2*hintsHeight - 4, Width: ctr.BottomWidth - 8, Height: 2 * hintsHeight}
	window(drawing, bar)
	if session.status != "" {
		pixelText(drawing, fitted(session.status, labelCharacters), bar.X+10, bar.Y+2, pixel.danger)
	}
	if math.Mod(session.now*blinksPerSecond, 1) < 0.65 {
		hints(drawing, bar, "A FLY   START MENU")
		return
	}
	hints(drawing, bar, "        START MENU")
}

// arrow draws a pixel triangle pointing left (-1) or right (1), shipArrowWidth wide.
func arrow(drawing *draw2d.List, x, y float32, direction int) {
	const height = 2*shipArrowWidth - 1
	for row := range height {
		width := float32(shipArrowWidth - int(abs(float32(row-shipArrowWidth+1))))
		left := x
		if direction < 0 {
			left = x + shipArrowWidth - width
		}
		drawing.Rectangle(draw2d.Rect{X: left, Y: y + float32(row), Width: width, Height: 1}, 0, pixel.gold, draw2d.Color{}, 0)
	}
}

// centredText draws a line in the middle of the screen's width, its baseline at y.
func (session *session) centredText(drawing *draw2d.List, text string, y, size float32, weight draw2d.Weight, colour draw2d.Color) {
	drawing.Text(text, (ctr.BottomWidth-session.textWidth(text, size, weight))/2, y, size, weight, colour)
}

func (session *session) textWidth(text string, size float32, weight draw2d.Weight) float32 {
	return session.measurer.MeasureText(text, size, weight, 1).Width
}

// shipNameCharacters is how much of a ship's name fits between its arrows.
const shipNameCharacters = 13

// shipLabel is a ship's name shortened to fit, with "My" before the player's own designs.
func shipLabel(choice shipChoice) string {
	label := fitted(choice.name, shipNameCharacters)
	if choice.own {
		return "My " + label
	}
	return label
}

func lifeLine(life, maximum float64) string {
	return "Life " + strconv.Itoa(int(life)) + " / " + strconv.Itoa(int(maximum))
}
