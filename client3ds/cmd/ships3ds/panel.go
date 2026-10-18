//go:build nintendo3ds

package main

import (
	"sort"
	"strconv"

	"github.com/akrck02/go-renderer/ctr"
	ui "github.com/akrck02/go-ui"
)

// build lays out the touch screen: the connection, our life and score, and everyone's score.
func (session *session) build(context *ui.Context) {
	state := session.game
	options := ui.PanelOptions{Dock: ui.DockLeft, Width: ui.Fixed(ctr.BottomWidth), Height: ui.Grow()}
	context.PanelWith("Ships 3D", options, func() {
		context.Caption(session.status)
		if state.Problem != "" {
			context.Caption("Server: " + state.Problem)
		}
		if session.client == nil && !state.Welcomed {
			session.buildServerChoice(context)
			return
		}
		if !state.Welcomed {
			return
		}
		lifeShare := float32(state.Player.Life / state.Settings.MaxLife)
		context.Row(func() {
			context.Text("Life", ui.TextStyle{})
			context.Spacer()
			context.Text(strconv.Itoa(int(state.Player.Life))+" / "+strconv.Itoa(int(state.Settings.MaxLife)), ui.TextStyle{})
		})
		context.Slider("life", &lifeShare, 0, 1) // shown only: the value is put back every frame
		if !state.Alive {
			context.Caption("Destroyed. Press A to fly again.")
		}
		context.Caption("Players")
		context.Text(scoreLine(state.Player.Name+" (you)", state.Player.Kills, state.Player.Deaths), ui.TextStyle{})
		for _, line := range session.remoteLines() {
			context.Text(line, ui.TextStyle{})
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

// buildServerChoice shows the server to connect to, a button to type another one, and Connect.
func (session *session) buildServerChoice(context *ui.Context) {
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
}
