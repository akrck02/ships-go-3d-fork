//go:build nintendo3ds

package main

import (
	"sort"
	"strconv"

	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"

	"ships3d/client3ds/profile"
)

// boldWeight is the heavy text of titles and names.
const boldWeight = draw2d.Bold

// build lays out the touch screen for what it shows now.
func (session *session) build(context *ui.Context) {
	switch {
	case session.confirming != noChange:
		session.buildPanel(context, session.buildRestartWarning)
	case session.screen == titleScreen:
		session.buildTitle(context)
	case session.screen == enteringScreen:
		session.buildPanel(context, session.buildEntering)
	case session.menuOpen:
		session.buildPanel(context, session.buildMenu)
	case session.screen == shipScreen:
		session.buildShipChoice(context)
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
		if session.status != "" {
			context.Caption(session.status)
		}
		if session.game.Problem != "" {
			context.Caption("Server: " + session.game.Problem)
		}
		content(context)
	})
}

// centred is a box filling the touch screen with its content in the middle.
func centred(context *ui.Context, content func()) {
	layout := ui.Layout{Direction: ui.Vertical, Width: ui.Fixed(ctr.BottomWidth), Height: ui.Fixed(ctr.BottomHeight), AlignX: ui.Center, AlignY: ui.Center, Gap: 12}
	context.Box(layout, content)
}

// buildTitle is the first screen's touch screen: how to go on; the name is on the top screen.
func (session *session) buildTitle(context *ui.Context) {
	centred(context, func() {
		context.Text("Touch the screen", ui.TextStyle{Size: bigSize, Weight: boldWeight})
		context.Caption("or press A to start")
	})
}

// buildEntering says the server is being reached, or why it couldn't be, with a way out.
func (session *session) buildEntering(context *ui.Context) {
	if session.asking == askEnter {
		return // the status says what is happening
	}
	context.Row(func() {
		if context.IconButton("refresh-cw", "Retry") {
			session.enter()
		}
		if context.IconButton("settings", "Server and account") {
			session.menuOpen, session.screen = true, shipScreen
			session.tab, session.tabIndex = accountTab, 0
		}
	})
}

// buildShipChoice is the ship screen's touch screen, over the turning ship: the name between
// arrows at the top, Fly at the bottom.
func (session *session) buildShipChoice(context *ui.Context) {
	layout := ui.Layout{Direction: ui.Vertical, Width: ui.Fixed(ctr.BottomWidth), Height: ui.Fixed(ctr.BottomHeight), AlignX: ui.Center,
		Padding: ui.Padding{Top: shipChoiceMargin, Bottom: shipChoiceMargin}, Gap: 4}
	context.Box(layout, func() {
		// The name between two marks that say the cross (or a touch on them) changes it: a row
		// as wide as the screen, its content centred.
		row := ui.Layout{Direction: ui.Horizontal, Width: ui.Fixed(ctr.BottomWidth), AlignX: ui.Center, AlignY: ui.Center, Gap: shipMarkGap}
		context.Box(row, func() {
			shipMark(context, "chevron-left")
			context.Text(shipLabel(session.ships[session.ship]), ui.TextStyle{Size: bigSize, Weight: boldWeight})
			shipMark(context, "chevron-right")
		})
		context.Caption(strconv.Itoa(session.ship+1) + " of " + strconv.Itoa(len(session.ships)) + "  ·  " + session.playerName())
		if session.status != "" {
			context.Caption(session.status)
		}
		context.Spacer() // the turning ship shows here
		if context.IconButton("rocket", "Fly") {
			session.connect()
		}
		context.Caption("A flies  ·  START: options")
	})
}

// shipChoiceMargin is the room above the arrows and below Fly, in pixels.
const shipChoiceMargin = 8

// buildRestartWarning asks before a change of server or account, which starts the game again.
func (session *session) buildRestartWarning(context *ui.Context) {
	context.Text("The game will restart", ui.TextStyle{Weight: boldWeight})
	context.Caption("You will leave the game and connect again.")
	context.Row(func() {
		if context.IconButton("rotate-ccw", "Restart") {
			session.confirm()
		}
		if context.IconButton("x", "Cancel") {
			session.confirming = noChange
		}
	})
}

// shipMarkGap is the room between the ship's name and its marks, in pixels.
const shipMarkGap = 10

// shipMark is a mark beside the ship's name, not a button: the cross changes the ship, and so
// does a touch on either side of the name (pressOnShipScreen).
func shipMark(context *ui.Context, icon string) {
	context.IconSized(icon, shipMarkSize, context.Theme.Primary)
}

// shipMarkSize is the marks' size, in pixels.
const shipMarkSize = 16

// shipNameCharacters is how much of a ship's name fits between its arrows.
const shipNameCharacters = 11

// shipLabel is a ship's name shortened to fit, with "My" before the player's own designs.
func shipLabel(choice shipChoice) string {
	label := []rune(choice.name)
	if len(label) > shipNameCharacters {
		label = append(label[:shipNameCharacters-1], '…')
	}
	if choice.own {
		return "My " + string(label)
	}
	return string(label)
}

// The tabs of the options, by name.
const (
	gameTab     = "Game"
	figuresTab  = "Figures"
	settingsTab = "Settings"
	accountTab  = "Account"
	controlsTab = "Controls"
)

// The options' tabs on the ship screen, and in the game.
var (
	shipMenuTabs = []string{accountTab, settingsTab, controlsTab}
	gameMenuTabs = []string{gameTab, figuresTab, settingsTab, accountTab, controlsTab}
)

// buildTabs shows tabs and the chosen one's content; a tab asked for by name (session.tab) is
// chosen when the list has it.
func (session *session) buildTabs(context *ui.Context, name string, labels []string, content map[string]func()) {
	selected := min(session.tabIndex, len(labels)-1)
	for index, label := range labels {
		if label == session.tab {
			selected = index
		}
	}
	context.Tabs(name, &selected, labels...)
	session.tabIndex, session.tab = selected, labels[selected]
	content[session.tab]()
}

// buildMenu is the options, opened with START on the ship screen or in the game.
func (session *session) buildMenu(context *ui.Context) {
	labels := shipMenuTabs
	if session.screen == gameScreen {
		labels = gameMenuTabs
	}
	session.buildTabs(context, "menu tabs", labels, map[string]func(){
		gameTab:     func() { session.buildGame(context) },
		figuresTab:  func() { session.buildFigures(context) },
		settingsTab: func() { session.buildSettings(context) },
		accountTab:  func() { session.buildAccount(context) },
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
		if context.IconButton("log-out", "Ships") {
			session.leave("")
		}
		if context.IconButton("power", "Quit") {
			session.quitting = true
		}
	})
}

// buildAccount shows the server and the account, each with its change, which restarts the game.
func (session *session) buildAccount(context *ui.Context) {
	context.Caption("Server")
	context.Row(func() {
		context.Text(session.profile.Server, ui.TextStyle{})
		context.Spacer()
		if context.IconButton("keyboard", "Change") {
			session.ask(changeServer)
		}
	})
	context.Caption("Account")
	context.Row(func() {
		if session.account.Token == "" {
			context.Text("Guest", ui.TextStyle{})
			context.Spacer()
			if context.IconButton("log-in", "Log in") {
				session.ask(changeLogin)
			}
			return
		}
		context.Text(session.account.Username, ui.TextStyle{Weight: boldWeight})
		context.Spacer()
		if context.IconButton("log-out", "Log out") {
			session.ask(changeLogout)
		}
	})
	if context.IconButton("chevron-left", "Back") {
		session.menuOpen = false
	}
}

// buildSettings offers how the ship flies and the radar shows; each change is remembered.
func (session *session) buildSettings(context *ui.Context) {
	settings := session.profile.Settings
	changed := context.Toggle("Up raises the nose", &settings.InvertPitch)
	threeDimensions := !settings.FlatRadar
	if context.Toggle("3D radar", &threeDimensions) {
		settings.FlatRadar, changed = !threeDimensions, true
	}
	if context.Toggle("Steer by tilting", &settings.Gyroscope) {
		session.tilt.Centre() // from where the console is held now
		changed = true
	}
	context.Row(func() {
		context.Text("Fine aim", ui.TextStyle{})
		context.Spacer()
		context.Text(strconv.Itoa(int(settings.Precision*100+0.5))+" %", ui.TextStyle{})
	})
	if context.SliderStep("precision", &settings.Precision, 0, 1, 0.1) {
		changed = true
	}
	context.Caption("Fine aim: the circle pad softer near its centre. L centres the tilt.")
	if changed {
		session.remember(func(remembered *profile.Profile) { remembered.Settings = settings })
	}
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
