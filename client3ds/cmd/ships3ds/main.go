//go:build nintendo3ds

// Ships 3D on the Nintendo 3DS: it joins a game of the server in this repository as a guest, flies
// a built-in ship and shows the others in 3D on the top screen, with the score on the touch
// screen. The server's address and the ship are set when building:
//
//	tinygo build -target=nintendo3ds -ldflags "-X main.host=192.168.1.20 -X main.port=3000" -o ships3ds.3dsx ./cmd/ships3ds
//
// It opens on a title screen; touching it leads to the server's address, typed on the system
// keyboard. In the game the touch screen is a map of the ships around, turning with ours.
//
// Circle pad: nose up and down, turn. L and R: roll. X and B: faster and slower. A: fire, and fly
// again after dying. Up and down on the cross: the map's range. START: the options (leave the
// game, quit).
package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/akrck02/go-3ds/app"
	"github.com/akrck02/go-3ds/bottom"
	"github.com/akrck02/go-3ds/keyboard"
	"github.com/akrck02/go-3ds/network"
	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/models"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/game"
	"ships3d/client3ds/protocol"
	"ships3d/client3ds/radar"
	"ships3d/client3ds/websocket"
)

// Set with -ldflags -X: where the server is, and what to fly.
var (
	host   = "192.168.1.2"
	port   = "3000"
	shipId = "default:arrow"
	name   = "3DS"
)

// websocketPath is where the server takes game connections (controllers/game.go).
const websocketPath = "/ws"

func main() {
	screen, err := ctr.Open()
	if err != nil {
		println(err.Error())
		return
	}
	defer screen.Close()
	panel, err := bottom.New(bottom.Theme())
	if err != nil {
		println(err.Error())
		return
	}
	renderer, err := ctr.NewSceneRenderer(screen)
	if err != nil {
		println(err.Error())
		return
	}

	session := newSession()
	world := newWorld()
	start := time.Now()
	view := models.Camera{Far: cameraFar, Near: cameraNear}
	app.RunWithStart(func(frame app.Frame) bool {
		now := time.Since(start).Seconds()
		if !session.press(frame.Pressed) {
			return false
		}
		session.exchange(frame, now)
		world.show(session.game, &view)

		ctr.BeginFrame()
		separation := ctr.Slider3D() * eyeSeparation
		renderer.PaintStereo(world.scene, view, separation, focalDistance)
		panel.Paint(frame.Input, session.build)
		ctr.EndFrame()
		if session.typing {
			session.typeServer()
		}
		return true
	})
	session.close()
}

// screen is what the touch screen shows.
type screen int

const (
	titleScreen  screen = iota // "Ships 3D", touch to start
	serverScreen               // the server's address and Connect
	gameScreen                 // the map, while connected
)

// session is the game, its connection to the server and what the touch screen shows.
type session struct {
	game       *game.Game
	connection *network.Connection
	client     *websocket.Client
	joined     bool
	server     string // host:port, typed on the system keyboard
	typing     bool   // the player asked for the keyboard; it opens after this frame
	status     string // what the touch screen says about the connection
	screen     screen
	menuOpen   bool // the options, over the map
	quitting   bool
	mapRange   int // index in radar.Ranges
}

// maximumServerLength fits a name or an address and its port.
const maximumServerLength = 64

func newSession() *session {
	return &session{game: game.New(), server: host + ":" + port, status: "Choose the server and touch Connect", mapRange: radar.DefaultRange}
}

// typeServer opens the system keyboard for the server's address. It runs between frames: the
// keyboard takes both screens until the player is done.
func (session *session) typeServer() {
	session.typing = false
	session.server, _ = keyboard.Ask(keyboard.Numbers, "Server address:port", session.server, maximumServerLength)
}

// connect opens the connection and starts the websocket handshake.
func (session *session) connect() {
	address, portText, found := strings.Cut(session.server, ":")
	if !found {
		portText = port
	}
	portNumber, err := strconv.Atoi(portText)
	if err != nil {
		session.status = "The port " + portText + " is not a number"
		return
	}
	session.connection, err = network.Dial(address, portNumber)
	if err != nil {
		session.status = err.Error()
		return
	}
	session.client, err = websocket.Connect(session.connection, address+":"+portText, websocketPath, nil)
	if err != nil {
		session.status = err.Error()
		return
	}
	session.status = "Connecting to " + address + ":" + portText + "…"
	session.screen = gameScreen
}

// press acts on the buttons pressed this frame that change screens, and tells whether to go on.
func (session *session) press(pressed ctr.Buttons) bool {
	if session.quitting {
		return false
	}
	switch session.screen {
	case titleScreen:
		if pressed&(ctr.ButtonTouch|ctr.ButtonA) != 0 {
			session.screen = serverScreen
		}
		return pressed&ctr.ButtonStart == 0
	case serverScreen:
		return pressed&ctr.ButtonStart == 0
	}
	if pressed&ctr.ButtonStart != 0 {
		session.menuOpen = !session.menuOpen
	}
	if pressed&ctr.ButtonUp != 0 {
		session.mapRange = max(session.mapRange-1, 0)
	}
	if pressed&ctr.ButtonDown != 0 {
		session.mapRange = min(session.mapRange+1, len(radar.Ranges)-1)
	}
	return true
}

// leave closes the connection and goes back to the server's address.
func (session *session) leave(status string) {
	session.close()
	session.client, session.connection = nil, nil
	session.game, session.joined, session.menuOpen = game.New(), false, false
	session.status, session.screen = status, serverScreen
}

// exchange reads what the server sent, moves the game on with the controls and sends the result.
func (session *session) exchange(frame app.Frame, now float64) {
	if session.client == nil {
		return
	}
	messages, err := session.client.Receive()
	for _, text := range messages {
		message, states, readErr := protocol.Read(text)
		if readErr != nil {
			println("unreadable message:", readErr.Error())
			continue
		}
		session.game.Handle(message, states, now)
	}
	if err != nil {
		session.leave("Disconnected: " + err.Error())
		return
	}
	if !session.client.Open() {
		return
	}
	if !session.joined {
		session.joined = true
		session.status = "Joining…"
		session.send(protocol.Join(shipId, name))
	}
	if session.game.Welcomed {
		session.status = "Playing on " + session.server
	}
	controls, firing := controlsOf(frame.Reading), frame.Reading.Held&ctr.ButtonA != 0
	if session.menuOpen {
		controls, firing = flight.Controls{}, false // the ship flies on straight while the options are open
	}
	for _, text := range session.game.Step(controls, firing, firing, frame.Seconds, now) {
		session.send(text)
	}
}

func (session *session) send(text string) {
	if session.client == nil {
		return // left on an earlier message of this frame
	}
	if err := session.client.Send(text); err != nil {
		session.leave("Disconnected: " + err.Error())
	}
}

func (session *session) close() {
	if session.client != nil {
		_ = session.client.Close() // leaving anyway; the server notices the socket close too
	}
	if session.connection != nil {
		_ = session.connection.Close() // as above
	}
	network.Stop()
}
