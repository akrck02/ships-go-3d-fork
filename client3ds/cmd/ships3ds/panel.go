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

// buildServerChoice shows the server to connect to, a button to type another one, and Connect;
// and the controls, in their own tab.
func (session *session) buildServerChoice(context *ui.Context) {
	context.Tabs("server tabs", &session.tab, "Server", "Controls")
	if session.tab == 1 {
		buildControls(context)
		return
	}
	context.Caption("Server")
	context.Text(session.server, ui.TextStyle{})
	context.Row(func() {
		if context.IconButton("keyboard", "Change") {
			session.typing = true
		}
		if context.IconButton("plug", "Connect") {
			session.connect()
		}
	})
	context.Caption("START quits")
}

// buildMenu is the options, opened with START during a game.
func (session *session) buildMenu(context *ui.Context) {
	context.Tabs("menu tabs", &session.tab, "Game", "Controls")
	if session.tab == 1 {
		buildControls(context)
		return
	}
	state := session.game
	context.Text(lifeLine(state.Player.Life, state.Settings.MaxLife), ui.TextStyle{})
	context.Caption("Players")
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
