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
	address    [4]float32 // the server's IPv4 address, chosen on the touch screen
	status     string     // what the touch screen says about the connection
}

func newSession() *session {
	session := &session{game: game.New(), status: "Choose the server and touch Connect"}
	parts := strings.Split(host, ".")
	for index := 0; index < len(parts) && index < len(session.address); index++ {
		number, _ := strconv.Atoi(parts[index]) // a name instead of numbers just starts at 0
		session.address[index] = float32(number)
	}
	return session
}

// serverHost is the address chosen on the touch screen.
func (session *session) serverHost() string {
	parts := make([]string, len(session.address))
	for index, number := range session.address {
		parts[index] = strconv.Itoa(int(number))
	}
	return strings.Join(parts, ".")
}

// connect opens the connection and starts the websocket handshake.
func (session *session) connect() {
	address := session.serverHost()
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		session.status = "The port " + port + " is not a number"
		return
	}
	session.connection, err = network.Dial(address, portNumber)
	if err != nil {
		session.status = err.Error()
		return
	}
	session.client, err = websocket.Connect(session.connection, address+":"+port, websocketPath, nil)
	if err != nil {
		session.status = err.Error()
		return
	}
	session.status = "Connecting to " + address + ":" + port + "…"
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
