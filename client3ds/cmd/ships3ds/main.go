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
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/akrck02/go-3ds/app"
	"github.com/akrck02/go-3ds/bottom"
	"github.com/akrck02/go-3ds/keyboard"
	"github.com/akrck02/go-3ds/network"
	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/models"

	"ships3d/client3ds/account"
	"ships3d/client3ds/figures"
	"ships3d/client3ds/flight"
	"ships3d/client3ds/game"
	"ships3d/client3ds/protocol"
	"ships3d/client3ds/radar"
	"ships3d/client3ds/tilt"
	"ships3d/client3ds/websocket"
)

// Set with -ldflags -X: where the server is, and the player's name.
var (
	host = "192.168.1.2"
	port = "3000"
	name = "3DS"
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
		frameStart := time.Now()
		now := frameStart.Sub(start).Seconds()
		if !session.press(frame.Pressed) {
			return false
		}
		session.exchange(frame, now)
		networkTime := time.Since(frameStart)
		world.show(session.game, &view, frame.Seconds)

		ctr.BeginFrame()
		separation := ctr.Slider3D() * eyeSeparation
		renderer.PaintStereo(world.scene, view, separation, focalDistance)
		panel.Paint(frame.Input, session.build)
		session.measure(frame, renderer, time.Since(frameStart), networkTime, now)
		ctr.EndFrame()
		session.askWithKeyboard()
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
	asking     asking // what the keyboard is to be opened for, after this frame
	account    account.Session
	email      string       // the last one typed, offered again
	ships      []shipChoice // what the Ship arrows go through
	status     string       // what the touch screen says about the connection
	screen     screen
	menuOpen   bool   // the options, over the map
	tab        string // the chosen tab of the server screen or the options
	tabIndex   int
	quitting   bool
	mapRange   int // index in radar.Ranges
	scanner    *radar.Scanner
	now        float64               // seconds, this frame
	blips      map[string]radar.Blip // where the radar sees each ship this frame, reused
	figures    figures.Counter
	reader     protocol.Reader
	ship       int // index in ships
	options    options
	tilt       tilt.Tilt
	gyroscope  app.Gyroscope // this frame's, for the Figures tab
	heapBytes  uint64        // the heap's size, read now and then
	memoryAt   float64       // when it was read, in seconds
}

func newSession() *session {
	return &session{game: game.New(), server: host + ":" + port, status: "Choose the server and touch Connect", mapRange: radar.DefaultRange,
		scanner: radar.NewScanner(), options: defaultOptions, blips: map[string]radar.Blip{}, ships: guestShips}
}

// asking is what the system keyboard is to be opened for.
type asking int

const (
	askNothing asking = iota
	askServer         // the server's address
	askLogin          // an email and a password
)

// Most characters the keyboard takes for each answer.
const (
	maximumServerLength   = 64 // a name or an address and its port
	maximumEmailLength    = 100
	maximumPasswordLength = 64
)

// askWithKeyboard opens the system keyboard if the player asked for it this frame. It runs
// between frames: the keyboard takes both screens until the player is done.
func (session *session) askWithKeyboard() {
	asked := session.asking
	session.asking = askNothing
	switch asked {
	case askServer:
		server, confirmed := keyboard.Ask(keyboard.Numbers, "Server address:port", session.server, maximumServerLength)
		if confirmed {
			session.server = server
			session.refreshShips()
		}
	case askLogin:
		session.logIn()
	}
}

// serverAddress splits what was typed into the address and the port, which is 3000 when left out.
func (session *session) serverAddress() (address string, portNumber int, err error) {
	address, portText, found := strings.Cut(session.server, ":")
	if !found {
		portText = port
	}
	portNumber, err = strconv.Atoi(portText)
	if err != nil {
		return "", 0, fmt.Errorf("the port %q is not a number", portText)
	}
	return address, portNumber, nil
}

// api is the server's HTTP side, for logging in and the list of ships.
func (session *session) api() (account.Server, error) {
	address, portNumber, err := session.serverAddress()
	if err != nil {
		return account.Server{}, err
	}
	dial := func() (account.Connection, error) { return network.Dial(address, portNumber) }
	return account.Server{Dial: dial, Host: address + ":" + strconv.Itoa(portNumber)}, nil
}

// logIn asks for an email and a password, logs in, and offers the player's own ships too.
func (session *session) logIn() {
	email, confirmed := keyboard.Ask(keyboard.Text, "Email", session.email, maximumEmailLength)
	if !confirmed {
		return
	}
	session.email = email
	password, confirmed := keyboard.Ask(keyboard.Password, "Password", "", maximumPasswordLength)
	if !confirmed {
		return
	}
	server, err := session.api()
	if err != nil {
		session.status = err.Error()
		return
	}
	logged, err := server.LogIn(email, password)
	if errors.Is(err, account.ErrRefused) {
		session.status = "Wrong email or password"
		return
	}
	if err != nil {
		session.status = "Couldn't log in: " + err.Error()
		return
	}
	session.account = logged
	session.status = "Logged in as " + logged.Username
	session.refreshShips()
}

// refreshShips asks the server which ships it has for this player, so the arrows offer only
// those: a server running an older version has fewer built-in ships. It waits for the answer, a
// few seconds at most; it runs after the keyboard, or when the player touches Connect.
func (session *session) refreshShips() bool {
	server, err := session.api()
	if err != nil {
		session.status = err.Error()
		return false
	}
	ships, err := server.Ships(session.account)
	if err != nil {
		session.status = "The server's ships didn't load: " + err.Error()
		return false
	}
	session.offer(ships)
	return true
}

// offer makes the Ship arrows go through the built-in ships, then the player's own, keeping the
// chosen one when the server has it.
func (session *session) offer(ships account.Ships) {
	chosen := session.ships[session.ship].id
	session.ships = session.ships[:0:0]
	for _, ship := range ships.Defaults {
		session.ships = append(session.ships, shipChoice{id: ship.Id, name: ship.Name})
	}
	for _, ship := range ships.Own {
		session.ships = append(session.ships, shipChoice{id: ship.Id, name: ship.Name, own: true})
	}
	if len(session.ships) == 0 {
		session.ships = guestShips
	}
	session.ship = 0
	for index, choice := range session.ships {
		if choice.id == chosen {
			session.ship = index
		}
	}
}

// offers tells whether the arrows' list holds a ship.
func (session *session) offers(id string) bool {
	for _, choice := range session.ships {
		if choice.id == id {
			return true
		}
	}
	return false
}

// logOut ends the session on the server and goes back to the built-in ships.
func (session *session) logOut() {
	if server, err := session.api(); err == nil {
		// Leaving anyway: a session the server keeps a little longer does no harm.
		_ = server.LogOut(session.account)
	}
	session.account = account.Session{}
	session.status = "Logged out"
	session.refreshShips()
}

// connect opens the connection and starts the websocket handshake, with the session's cookie
// when logged in.
func (session *session) connect() {
	chosen := session.ships[session.ship]
	if session.refreshShips() && !session.offers(chosen.id) {
		session.status = "This server has no " + chosen.name + ": choose another ship"
		return
	}
	address, portNumber, err := session.serverAddress()
	if err != nil {
		session.status = err.Error()
		return
	}
	session.connection, err = network.Dial(address, portNumber)
	if err != nil {
		session.status = err.Error()
		return
	}
	var header map[string]string
	if session.account.Token != "" {
		header = map[string]string{"Cookie": session.account.Cookie()}
	}
	hostPort := address + ":" + strconv.Itoa(portNumber)
	session.client, err = websocket.Connect(session.connection, hostPort, websocketPath, header)
	if err != nil {
		session.status = err.Error()
		return
	}
	session.status = "Connecting to " + hostPort + "…"
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
	session.scanner = radar.NewScanner()
	session.status, session.screen = status, serverScreen
}

// exchange reads what the server sent, moves the game on with the controls and sends the result.
func (session *session) exchange(frame app.Frame, now float64) {
	if session.client == nil {
		return
	}
	messages, err := session.client.Receive()
	for _, text := range messages {
		message, states, readErr := session.reader.Read(text)
		if readErr != nil {
			println(readErr.Error()) // the debug log; one bad message doesn't end the game
			continue
		}
		session.figures.Received(len(text))
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
		session.send(protocol.Join(session.ships[session.ship].id, session.playerName()))
	}
	if session.game.Welcomed {
		session.status = "Playing on " + session.server
	}
	tiltOf(&session.tilt, frame)
	if frame.Pressed&ctr.ButtonL != 0 {
		session.tilt.Centre()
	}
	firing := frame.Reading.Held&fireButtons != 0
	controls := controlsOf(frame, session.options, session.tilt)
	if session.menuOpen {
		controls, firing = flight.Controls{}, false // the ship flies on straight while the options are open
	}
	for _, text := range session.game.Step(controls, firing, firing, frame.Seconds, now) {
		session.send(text)
	}
	session.scan(now)
}

// scan turns the map's sweep and lets it see the ships it passes over.
func (session *session) scan(now float64) {
	state := session.game
	session.now = now
	blips := session.blips
	clear(blips)
	for id, remote := range state.Remotes {
		if !remote.Player.Dead {
			blips[id] = radar.Place(remote.Shown.Position, state.Own.Position, state.Own.Rotation, radar.Ranges[session.mapRange])
		}
	}
	session.scanner.Update(now, blips)
}

func (session *session) send(text string) {
	if session.client == nil {
		return // left on an earlier message of this frame
	}
	session.figures.Sent(len(text))
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

// measure counts the frame for the Figures tab; processor is how long the frame took before
// waiting for the screen.
func (session *session) measure(frame app.Frame, renderer *ctr.SceneRenderer, processor, network time.Duration, now float64) {
	statistics := renderer.Statistics(processor)
	if session.menuOpen && session.tab == figuresTab && now-session.memoryAt >= memoryEverySeconds {
		// Reading the heap walks it, which takes most of a frame: only while shown, once a second.
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		session.heapBytes, session.memoryAt = memory.HeapAlloc, now
	}
	delay := 0.0
	if session.game.SnapshotAt > 0 {
		delay = now - session.game.SnapshotAt
	}
	session.gyroscope = frame.Gyroscope
	session.figures.AddFrame(figures.Frame{
		Seconds:               frame.Seconds,
		ProcessorMilliseconds: float32(processor.Seconds() * 1000),
		NetworkMilliseconds:   float32(network.Seconds() * 1000),
		DrawingMilliseconds:   float32((processor - network).Seconds() * 1000),
		GPUMilliseconds:       ctr.DrawingTime(),
		Triangles:             statistics.Triangles,
		DrawCalls:             statistics.DrawCalls,
		Bakes:                 renderer.Bakes(),
		HeapBytes:             session.heapBytes,
		Players:               len(session.game.Remotes) + 1,
		SnapshotDelay:         delay,
	})
}

// shipChoice is a ship the player may fly: its id for the server, its name, and whether it is
// one of the player's own designs.
type shipChoice struct {
	id, name string
	own      bool
}

// playerName is the logged-in player's name, or the guest's.
func (session *session) playerName() string {
	if session.account.Username != "" {
		return session.account.Username
	}
	return name
}

// guestShips are what the arrows offer before the server has said which ships it has: the
// built-in ships every version of it has (game/defaultShips.go).
var guestShips = []shipChoice{
	{id: "default:arrow", name: "Arrow"},
	{id: "default:hammer", name: "Hammer"},
	{id: "default:saucer", name: "Saucer"},
}

// options are the player's choices in the Settings tab.
type options struct {
	invertPitch bool    // up on the circle pad raises the nose, instead of a plane's way
	precision   float32 // 0 to 1: how much softer the circle pad is near its centre
	gyroscope   bool    // tilting the console steers too
}

var defaultOptions = options{precision: 0.5}

// memoryEverySeconds is how often the Figures tab reads the heap's size.
const memoryEverySeconds = 1
