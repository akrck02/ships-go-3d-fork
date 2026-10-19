//go:build nintendo3ds

package main

import (
	"sort"
	"strconv"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"
)

// boldWeight is the heavy text of titles and names.
const boldWeight = draw2d.Bold

// shipNameWidth is the room the chosen ship's name has between its arrows, in pixels.
const shipNameWidth = 96

// titleSize is the title screen's heading, in pixels.
const titleSize = 40

// build lays out the touch screen for what it shows now.
func (session *session) build(context *ui.Context) {
	switch {
	case session.screen == titleScreen:
		session.buildTitle(context)
	case session.screen == serverScreen:
		session.buildPanel(context, session.buildServerChoice)
	case session.menuOpen:
		session.buildPanel(context, session.buildMenu)
	case session.game.Welcomed:
		session.buildMap(context)
	default:
		session.buildPanel(context, func(context *ui.Context) {})
	}
}

// buildPanel is a panel over the whole screen with the connection's status and content below.
func (session *session) buildPanel(context *ui.Context, content func(context *ui.Context)) {
	options := ui.PanelOptions{Dock: ui.DockLeft, Width: ui.Fixed(ctr.BottomWidth), Height: ui.Grow()}
	context.PanelWith("Ships 3D", options, func() {
		context.Caption(session.status)
		if session.game.Problem != "" {
			context.Caption("Server: " + session.game.Problem)
		}
		content(context)
	})
}

// buildTitle is the first screen: the name, and how to go on.
func (session *session) buildTitle(context *ui.Context) {
	layout := ui.Layout{Direction: ui.Vertical, Width: ui.Fixed(ctr.BottomWidth), Height: ui.Fixed(ctr.BottomHeight), AlignX: ui.Center, AlignY: ui.Center, Gap: 12}
	context.Box(layout, func() {
		context.Text("Ships 3D", ui.TextStyle{Size: titleSize, Weight: boldWeight})
		context.Caption("Touch the screen or press A to start")
	})
}

// The tabs of the server screen and of the options, by name: both share the controls and the
// settings.
const (
	serverTab   = "Server"
	gameTab     = "Game"
	controlsTab = "Controls"
	figuresTab  = "Figures"
	settingsTab = "Settings"
)

var (
	serverTabs = []string{serverTab, settingsTab, controlsTab}
	menuTabs   = []string{gameTab, figuresTab, settingsTab, controlsTab}
)

// buildTabs shows tabs and the chosen one's content.
func (session *session) buildTabs(context *ui.Context, name string, labels []string, content map[string]func()) {
	selected := min(session.tabIndex, len(labels)-1)
	context.Tabs(name, &selected, labels...)
	session.tabIndex, session.tab = selected, labels[selected]
	content[session.tab]()
}

// buildServerChoice shows the server to connect to, the ship, and Connect; the settings and the
// controls in their own tabs.
func (session *session) buildServerChoice(context *ui.Context) {
	session.buildTabs(context, "server tabs", serverTabs, map[string]func(){
		serverTab:   func() { session.buildServer(context) },
		settingsTab: func() { session.buildSettings(context) },
		controlsTab: func() { buildControls(context) },
	})
}

func (session *session) buildServer(context *ui.Context) {
	context.Row(func() {
		context.Text(session.server, ui.TextStyle{})
		context.Spacer()
		if context.IconButton("keyboard", "Change") {
			session.typing = true
		}
	})
	// Arrows rather than a drop-down list: eight ships don't fit below it on the touch screen.
	context.Row(func() {
		context.Text("Ship", ui.TextStyle{})
		context.Spacer()
		if context.IconButton("chevron-left", "") {
			session.ship = (session.ship + len(guestShips) - 1) % len(guestShips)
		}
		context.Box(ui.Layout{Width: ui.Fixed(shipNameWidth), AlignX: ui.Center}, func() {
			context.Text(guestShips[session.ship].name, ui.TextStyle{Weight: boldWeight})
		})
		if context.IconButton("chevron-right", "") {
			session.ship = (session.ship + 1) % len(guestShips)
		}
	})
	if context.IconButton("plug", "Connect") {
		session.connect()
	}
	context.Caption("START quits")
}

// buildMenu is the options, opened with START during a game.
func (session *session) buildMenu(context *ui.Context) {
	session.buildTabs(context, "menu tabs", menuTabs, map[string]func(){
		gameTab:     func() { session.buildGame(context) },
		figuresTab:  func() { session.buildFigures(context) },
		settingsTab: func() { session.buildSettings(context) },
		controlsTab: func() { buildControls(context) },
	})
}

func (session *session) buildGame(context *ui.Context) {
	state := session.game
	context.Text(lifeLine(state.Player.Life, state.Settings.MaxLife), ui.TextStyle{})
	context.Text(scoreLine(state.Player.Name+" (you)", state.Player.Kills, state.Player.Deaths), ui.TextStyle{})
	for _, line := range session.remoteLines() {
		context.Text(line, ui.TextStyle{})
	}
	context.Row(func() {
		if context.IconButton("play", "Resume") {
			session.menuOpen = false
		}
		if context.IconButton("log-out", "Leave") {
			session.leave("Left the game")
		}
		if context.IconButton("power", "Quit") {
			session.quitting = true
		}
	})
}

// buildSettings offers how the circle pad flies.
func (session *session) buildSettings(context *ui.Context) {
	context.Toggle("Up raises the nose", &session.options.invertPitch)
	if context.Toggle("Steer by tilting", &session.options.gyroscope) {
		session.tilt.Centre() // from where the console is held now
	}
	context.Row(func() {
		context.Text("Fine aim", ui.TextStyle{})
		context.Spacer()
		context.Text(strconv.Itoa(int(session.options.precision*100+0.5))+" %", ui.TextStyle{})
	})
	context.SliderStep("precision", &session.options.precision, 0, 1, 0.1)
	context.Caption("Fine aim: the circle pad softer near its centre.")
	context.Caption("L centres the tilt where you hold the console.")
}

// remoteLines are the other players' scores, best first.
func (session *session) remoteLines() []string {
	type score struct {
		name          string
		kills, deaths int
	}
	scores := []score{}
	for _, remote := range session.game.Remotes {
		scores = append(scores, score{remote.Player.Name, remote.Player.Kills, remote.Player.Deaths})
	}
	sort.Slice(scores, func(first, second int) bool { return scores[first].kills > scores[second].kills })
	lines := make([]string, 0, len(scores))
	for _, entry := range scores {
		lines = append(lines, scoreLine(entry.name, entry.kills, entry.deaths))
	}
	return lines
}

func scoreLine(name string, kills, deaths int) string {
	return name + "   " + strconv.Itoa(kills) + " kills, " + strconv.Itoa(deaths) + " deaths"
}

func lifeLine(life, maximum float64) string {
	return "Life " + strconv.Itoa(int(life)) + " / " + strconv.Itoa(int(maximum))
}
