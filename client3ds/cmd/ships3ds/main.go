//go:build nintendo3ds

// Ships 3D on the Nintendo 3DS: it joins a game of the server in this repository as a guest, flies
// a built-in ship and shows the others in 3D on the top screen, with the score on the touch
// screen. The server's address and the ship are set when building:
//
//	tinygo build -target=nintendo3ds -ldflags "-X main.host=192.168.1.20 -X main.port=3000" -o ships3ds.3dsx ./cmd/ships3ds
//
// Circle pad: nose up and down, turn. L and R: roll. X and B: faster and slower. A: fire, and fly
// again after dying. START leaves.
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

	"ships3d/client3ds/game"
	"ships3d/client3ds/protocol"
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
	app.Run(func(frame app.Frame) bool {
		now := time.Since(start).Seconds()
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

// session is the game and its connection to the server.
type session struct {
	game       *game.Game
	connection *network.Connection
	client     *websocket.Client
	joined     bool
	server     string // host:port, typed on the system keyboard
	typing     bool   // the player asked for the keyboard; it opens after this frame
	status     string // what the touch screen says about the connection
}

// maximumServerLength fits a name or an address and its port.
const maximumServerLength = 64

func newSession() *session {
	return &session{game: game.New(), server: host + ":" + port, status: "Choose the server and touch Connect"}
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
		session.status = "Disconnected: " + err.Error()
		session.client = nil
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
		session.status = "Playing on " + host
	}
	held := frame.Reading.Held
	for _, text := range session.game.Step(controlsOf(frame.Reading), held&ctr.ButtonA != 0, held&ctr.ButtonA != 0, frame.Seconds, now) {
		session.send(text)
	}
}

func (session *session) send(text string) {
	if err := session.client.Send(text); err != nil {
		session.status = "Disconnected: " + err.Error()
		session.client = nil
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
