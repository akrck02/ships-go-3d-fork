//go:build nintendo3ds

// Ships 3D on the Nintendo 3DS: it joins a game of the server in this repository, as a guest or
// logged in, flies a ship and shows the others in 3D on the top screen, with a radar on the touch
// screen. It starts with the real server; another one is typed on the system keyboard, or set
// when building:
//
//	tinygo build -target=nintendo3ds -ldflags "-X main.server=192.168.1.20:3000" -o ships3ds.3dsx ./cmd/ships3ds
//
// The controls are listed in the Controls tab (controls.go).
package main

import (
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/akrck02/go-3ds/app"
	"github.com/akrck02/go-3ds/bottom"
	"github.com/akrck02/go-3ds/network"
	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/models"

	"ships3d/client3ds/account"
	"ships3d/client3ds/chase"
	"ships3d/client3ds/figures"
	"ships3d/client3ds/flight"
	"ships3d/client3ds/game"
	"ships3d/client3ds/profile"
	"ships3d/client3ds/protocol"
	"ships3d/client3ds/radar"
	"ships3d/client3ds/tilt"
	"ships3d/client3ds/websocket"
)

// Set with -ldflags -X: the server the game starts with, and a guest's name. The real server is
// typed with https://, so the game talks TLS to it.
var (
	server = "https://jonbul.ddns.net:3000"
	name   = "3DS"
)

// websocketPath is where the server takes game connections (controllers/game.go).
const websocketPath = "/ws"

func main() {
	// The socket service wants a megabyte in one piece: asked for first, before the fonts, the
	// painters and the scenes break C's memory up.
	if err := network.Start(); err != nil {
		println(err.Error()) // the debug log; entering the server says so again
	}
	screen, err := ctr.Open()
	if err != nil {
		println(err.Error())
		return
	}
	defer screen.Close()
	usePixelFonts()
	panel, err := bottom.New(spaceTheme())
	if err != nil {
		println(err.Error())
		return
	}
	renderer, err := ctr.NewSceneRenderer(screen)
	if err != nil {
		println(err.Error())
		return
	}
	dish, err := newRadarScene(screen)
	if err != nil {
		println(err.Error())
		return
	}
	preview, err := newPreviewScene(screen)
	if err != nil {
		println(err.Error())
		return
	}
	title, err := newTopTitle(panel.Screen(), spaceTheme())
	if err != nil {
		println(err.Error())
		return
	}

	session := newSession(loadProfile())
	session.log = newLogbook()
	session.log.add("Ships 3D for the 3DS starts; server " + session.profile.Server + ", logged in: " + strconv.FormatBool(session.account.Token != ""))
	session.log.connectRemote()
	world := newWorld()
	start := time.Now()
	view := models.Camera{Far: cameraFar, Near: cameraNear, FieldOfView: chase.FieldOfView}
	app.RunWithStart(func(frame app.Frame) bool {
		frameStart := time.Now()
		now := frameStart.Sub(start).Seconds()
		session.now = now
		if !session.press(frame) {
			return false
		}
		session.exchange(frame, now)
		networkTime := time.Since(frameStart)
		world.show(session.game, &view, frame.Seconds, now)
		heading, caption := session.topHeading()
		switch session.screen {
		case titleScreen, enteringScreen:
			world.showTitle(now, &view)
		case shipScreen:
			world.showTitle(now, &view)
			world.title.Hidden = true // the ship turns on the touch screen; the top shows its name
		}

		ctr.BeginFrame()
		if heading != "" {
			// One picture rather than two eyes: the words painted over it would show double in 3D.
			renderer.PaintOn(screen, world.scene, view)
			title.paint(screen, heading, caption)
		} else {
			separation := ctr.Slider3D() * eyeSeparation
			renderer.PaintStereo(world.scene, view, separation, focalDistance)
		}
		switch {
		case session.showsRadar3D():
			dish.paint(panel.Screen(), session.scanner, now)
			panel.PaintOver(frame.Input, session.build)
		case session.screen == shipScreen && !session.menuOpen && session.confirming == noChange:
			mesh, scale := session.previewShip(world)
			preview.paint(panel.Screen(), mesh, scale, now)
			panel.PaintOver(frame.Input, session.build)
		default:
			panel.Paint(frame.Input, session.build)
		}
		session.measure(frame, renderer, time.Since(frameStart), networkTime, now)
		ctr.EndFrame()
		session.betweenFrames()
		return true
	})
	session.close()
}

// screen is what the touch screen shows.
type screen int

const (
	titleScreen    screen = iota // "Ships 3D", touch to start
	enteringScreen               // logging in to the server and fetching its ships, or why that failed
	shipScreen                   // the ship to fly, turning on the top screen
	gameScreen                   // the radar, while connected
)

// session is the game, its connection to the server and what the touch screen shows.
type session struct {
	game       *game.Game
	connection *network.Connection
	client     *websocket.Client
	joined     bool
	profile    profile.Profile // what is remembered on the SD card
	account    account.Session
	ships      []shipChoice // what the ship screen's arrows go through
	ship       int          // index in ships
	status     string       // what the touch screen says about the connection
	screen     screen
	asking     asking // what to do between frames: the keyboard, or a wait for the network
	confirming change // a change of server or account waiting for "the game will restart"
	circleSide int    // where the circle pad was last frame on the ship screen: -1, 0 or 1
	menuOpen   bool   // the options, over the ship screen or the radar
	tab        string // the chosen tab of the options
	tabIndex   int
	quitting   bool
	mapRange   int // index in radar.Ranges
	scanner    *radar.Scanner
	now        float64               // seconds, this frame
	blips      map[string]radar.Blip // where the radar sees each ship this frame, reused
	figures    figures.Counter
	reader     protocol.Reader
	tilt       tilt.Tilt
	gyroscope  app.Gyroscope // this frame's, for the Figures tab
	heapBytes  uint64        // the heap's size, read now and then
	memoryAt   float64       // when it was read, in seconds
	log        *logbook
	logged     string  // the status last written to the log
	figuresAt  float64 // when figures were last logged
	skipped    int     // position updates skipped while the network was behind, since the last figures
}

func newSession(remembered profile.Profile) *session {
	return &session{game: game.New(), profile: remembered, mapRange: radar.DefaultRange,
		scanner: radar.NewScanner(), blips: map[string]radar.Blip{}, ships: guestShips,
		account: account.Session{Token: remembered.Token, Username: remembered.Username}}
}

// dial connects to the server, over TLS when it was typed with https://.
func dial(address account.Address) (*network.Connection, error) {
	if address.Secure {
		return network.DialSecure(address.Host, address.Port)
	}
	return network.Dial(address.Host, address.Port)
}

// api is the server's HTTP side, for logging in and the list of ships.
func (session *session) api() (account.Server, error) {
	address, err := account.ParseAddress(session.profile.Server)
	if err != nil {
		return account.Server{}, err
	}
	connect := func() (account.Connection, error) { return dial(address) }
	return account.Server{Dial: connect, Host: address.HostPort()}, nil
}

// connect opens the game's connection and starts the websocket handshake, with the session's
// cookie when logged in.
func (session *session) connect() {
	address, err := account.ParseAddress(session.profile.Server)
	if err != nil {
		session.status = err.Error()
		return
	}
	session.connection, err = dial(address)
	if err != nil {
		session.status = err.Error()
		return
	}
	var header map[string]string
	if session.account.Token != "" {
		header = map[string]string{"Cookie": session.account.Cookie()}
	}
	hostPort := address.HostPort()
	session.client, err = websocket.Connect(session.connection, hostPort, websocketPath, header)
	if err != nil {
		session.status = err.Error()
		return
	}
	session.status = "Connecting to " + hostPort + "…"
	session.screen = gameScreen
	session.remember(func(remembered *profile.Profile) { remembered.Ship = session.ships[session.ship].id })
}

// press acts on the buttons pressed this frame that change screens, and tells whether to go on.
func (session *session) press(frame app.Frame) bool {
	pressed := frame.Pressed
	if session.quitting {
		return false
	}
	switch session.screen {
	case titleScreen:
		if pressed&(ctr.ButtonTouch|ctr.ButtonA) != 0 {
			session.enter()
		}
		return pressed&ctr.ButtonStart == 0
	case enteringScreen:
		return true
	case shipScreen:
		session.pressOnShipScreen(pressed, frame.Reading)
		return true
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

// leave closes the game's connection and goes back to choosing a ship.
func (session *session) leave(status string) {
	session.close()
	session.client, session.connection = nil, nil
	session.game, session.joined, session.menuOpen = game.New(), false, false
	session.scanner = radar.NewScanner()
	session.status, session.screen = status, shipScreen
}

// exchange reads what the server sent, moves the game on with the controls and sends the result.
func (session *session) exchange(frame app.Frame, now float64) {
	if session.client == nil {
		return
	}
	if err := session.connection.Flush(); err != nil {
		session.leave("Disconnected: " + err.Error())
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
		if message.Type == protocol.TypeError {
			session.log.add("server says: " + message.Message)
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
		session.send(protocol.Join(session.ships[session.ship].id, session.playerName()))
	}
	if session.game.Welcomed {
		session.status = "Playing on " + session.profile.Server
	}
	tiltOf(&session.tilt, frame)
	if frame.Pressed&ctr.ButtonL != 0 {
		session.tilt.Centre()
	}
	firing := frame.Reading.Held&fireButtons != 0
	controls := controlsOf(frame, session.profile.Settings, session.tilt)
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

// stateBacklogBytes is how much may wait to be sent before position updates are skipped: each
// replaces the one before, so sending a stale one behind a stalled network only adds to the pile.
const stateBacklogBytes = 2048

// stateStart is how every position update begins (protocol.State).
const stateStart = `{"type":"state"`

func (session *session) send(text string) {
	if session.client == nil {
		return // left on an earlier message of this frame
	}
	if strings.HasPrefix(text, stateStart) && session.connection.Pending() > stateBacklogBytes {
		session.skipped++
		return
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

// shipChoice is a ship the player may fly: its id for the server, its name, whether it is one of
// the player's own designs, and its design, to show it turning.
type shipChoice struct {
	id, name string
	own      bool
	layers   []protocol.Layer
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

// showsRadar3D tells whether the touch screen shows the 3D radar now: in the game, with the
// options closed, unless the flat one was chosen.
func (session *session) showsRadar3D() bool {
	return session.screen == gameScreen && !session.menuOpen && session.game.Welcomed && !session.profile.Settings.FlatRadar
}

// memoryEverySeconds is how often the Figures tab reads the heap's size.
const memoryEverySeconds = 1
