//go:build nintendo3ds

package main

import (
	"errors"
	"math"
	"strconv"

	"github.com/akrck02/go-3ds/keyboard"
	"github.com/akrck02/go-3ds/storage"
	"github.com/akrck02/go-renderer/ctr"
	"github.com/akrck02/go-renderer/scene"

	"ships3d/client3ds/account"
	"ships3d/client3ds/game"
	"ships3d/client3ds/profile"
	"ships3d/client3ds/ship"
)

// The way through the game: the title, then entering the remembered server (logging in again
// with the remembered session), then the ship to fly, then the game. The server and the account
// change in the options, and changing them starts again from entering, after a warning.

// profileFolder and profileFile are where the profile is kept: sdmc:/3ds/ships3d/profile.json.
const (
	profileFolder = "ships3d"
	profileFile   = "profile.json"
)

// loadProfile reads what the last run remembered, or starts with the real server.
func loadProfile() profile.Profile {
	folder, err := storage.Open(profileFolder)
	if err != nil {
		println(err.Error()) // the debug log; the game goes on, remembering nothing
		return profile.New(server)
	}
	saved, err := folder.Load(profileFile)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		println(err.Error()) // as above
	}
	return profile.Read(saved, server)
}

// remember changes the profile and saves it on the SD card.
func (session *session) remember(change func(remembered *profile.Profile)) {
	change(&session.profile)
	folder, err := storage.Open(profileFolder)
	if err == nil {
		err = folder.Save(profileFile, session.profile.Bytes())
	}
	if err != nil {
		println(err.Error()) // the debug log: playing on matters more than remembering
	}
}

// asking is what to do between frames: open the system keyboard, which takes both screens, or
// wait for the server, so the screen first says what is happening.
type asking int

const (
	askNothing asking = iota
	askEnter          // log in again and fetch the ships
	askServer         // type the server's address
	askLogin          // type an email and a password
	askLogout         // end the session
)

// change is a change of server or account that restarts the game.
type change int

const (
	noChange change = iota
	changeServer
	changeLogin
	changeLogout
)

// Most characters the keyboard takes for each answer.
const (
	maximumServerLength   = 64 // a name or an address and its port
	maximumEmailLength    = 100
	maximumPasswordLength = 64
)

// betweenFrames does what was asked this frame and can't be done during one, and keeps the log.
func (session *session) betweenFrames() {
	session.keepLog()
	asked := session.asking
	session.asking = askNothing
	switch asked {
	case askEnter:
		session.logInAgainAndFetchShips()
	case askServer:
		session.typeServer()
	case askLogin:
		session.typeLogin()
	case askLogout:
		session.logOut()
	}
}

// enter goes from the title to the remembered server: the touch screen says so this frame, and
// the waiting happens between frames.
func (session *session) enter() {
	session.sounds.play(session.sounds.choose, 1)
	session.screen, session.asking = enteringScreen, askEnter
	session.status = "Connecting to " + session.profile.Server + "…"
}

// logInAgainAndFetchShips checks the remembered session with the server and fetches the ships it
// has for this player, then shows the ship screen; on a failure, the entering screen says why.
func (session *session) logInAgainAndFetchShips() {
	server, err := session.api()
	if err != nil {
		session.status = err.Error()
		return
	}
	if session.account.Token != "" {
		username, valid, err := server.Who(session.account)
		if err != nil {
			session.status = "Couldn't reach the server: " + err.Error()
			return
		}
		if !valid {
			session.forgetSession() // it expired: play as a guest, and say so on the ship screen
		} else {
			session.account.Username = username
		}
	}
	ships, err := server.Ships(session.account)
	if err != nil {
		session.status = "The server's ships didn't load: " + err.Error()
		return
	}
	session.offer(ships)
	session.screen, session.status = shipScreen, ""
	if session.account.Token == "" && session.profile.Email != "" {
		session.status = "Your session ended: log in again in the options"
	}
}

// offer makes the ship screen's arrows go through the built-in ships, then the player's own,
// starting at the one flown last.
func (session *session) offer(ships account.Ships) {
	session.ships = session.ships[:0:0]
	for _, ship := range ships.Defaults {
		session.ships = append(session.ships, shipChoice{id: ship.Id, name: ship.Name, layers: ship.Layers})
	}
	for _, ship := range ships.Own {
		session.ships = append(session.ships, shipChoice{id: ship.Id, name: ship.Name, own: true, layers: ship.Layers})
	}
	if len(session.ships) == 0 {
		session.ships = guestShips
	}
	session.ship = 0
	for index, choice := range session.ships {
		if choice.id == session.profile.Ship {
			session.ship = index
		}
	}
}

// circleSide is how far to one side the circle pad goes before it counts as a press.
const circleSide = 0.6

// shipRowHeight is how far down the touch screen the ship's name and its marks reach, in pixels.
const shipRowHeight = 56

// pressOnShipScreen goes through the ships with the cross or the circle pad, flies the chosen one
// with A, and opens the options with START.
func (session *session) pressOnShipScreen(pressed ctr.Buttons, reading ctr.Reading, input menuInput) {
	circleX, _ := reading.Circle()
	side := 0
	if circleX > circleSide {
		side = 1
	} else if circleX < -circleSide {
		side = -1
	}
	pushed := side != session.circleSide // pushed to a side this frame, not held there
	session.circleSide = side
	if pressed&ctr.ButtonStart != 0 {
		session.toggleMenu()
	}
	if session.menuOpen {
		session.pressPages(input)
		return
	}
	// A touch on the name's row: its left third goes back, its right third on.
	touchedRow := pressed&ctr.ButtonTouch != 0 && reading.TouchY < shipRowHeight
	touchedLeft := touchedRow && reading.TouchX < ctr.BottomWidth/3
	touchedRight := touchedRow && reading.TouchX > ctr.BottomWidth*2/3
	switch {
	case pressed&ctr.ButtonLeft != 0 || pushed && side < 0 || touchedLeft:
		session.previousShip()
	case pressed&ctr.ButtonRight != 0 || pushed && side > 0 || touchedRight:
		session.nextShip()
	case pressed&ctr.ButtonA != 0:
		session.connect()
	}
}

func (session *session) previousShip() {
	session.ship = (session.ship + len(session.ships) - 1) % len(session.ships)
	session.sounds.play(session.sounds.move, 1)
}

func (session *session) nextShip() {
	session.ship = (session.ship + 1) % len(session.ships)
	session.sounds.play(session.sounds.move, 1)
}

// toggleMenu opens or closes the options, with its sound.
func (session *session) toggleMenu() {
	session.menuOpen = !session.menuOpen
	if session.menuOpen {
		session.menu.opened = "" // the menu opens new, on its first page
		session.sounds.play(session.sounds.choose, 1)
		return
	}
	session.sounds.play(session.sounds.back, 1)
}

// previewShip is the chosen ship's mesh and the scale that makes it previewRadius big, for the
// ship screen to turn; no mesh for a ship the server hasn't described.
func (session *session) previewShip(world *world) (*scene.Mesh, float64) {
	choice := session.ships[session.ship]
	radius := ship.Radius(choice.layers)
	if radius <= 0 {
		return nil, 0
	}
	return world.shipMesh(choice.id, choice.layers, false), previewRadius / radius
}

// topHeading is what the top screen says over a still view: the title, or the ship being chosen;
// "" in the game.
func (session *session) topHeading() (heading, caption string) {
	switch session.screen {
	case titleScreen, enteringScreen:
		return "Ships 3D", "for Nintendo 3DS"
	case gameScreen:
		if session.game.Welcomed && !session.game.Alive {
			return "Destroyed", session.deathCaption()
		}
	case shipScreen:
		choice := session.ships[session.ship]
		caption = "Built-in ship"
		if choice.own {
			caption = "Your design"
		}
		return choice.name, caption
	}
	return "", ""
}

// ask is the options' way to change the server or the account: it first warns that the game
// will restart (confirm), then asks.
func (session *session) ask(wanted change) { session.confirming = wanted }

// confirm goes on with the change the player accepted the restart for.
func (session *session) confirm() {
	wanted := session.confirming
	session.confirming = noChange
	switch wanted {
	case changeServer:
		session.asking = askServer
	case changeLogin:
		session.asking = askLogin
	case changeLogout:
		session.asking = askLogout
	}
}

// restart leaves the game and enters the server again, with what changed.
func (session *session) restart(status string) {
	if session.client != nil {
		session.leave("")
	}
	session.menuOpen = false
	session.enter()
	session.status = status
}

func (session *session) typeServer() {
	typed, confirmed := keyboard.Ask(keyboard.Text, "Server: address:port, or https://name:port", session.profile.Server, maximumServerLength)
	if !confirmed || typed == session.profile.Server {
		return
	}
	if _, err := account.ParseAddress(typed); err != nil {
		session.status = err.Error()
		return
	}
	// A session belongs to the server it was made on.
	session.forgetSession()
	session.remember(func(remembered *profile.Profile) { remembered.Server = typed })
	session.restart("Connecting to " + typed + "…")
}

// typeLogin asks for an email and a password and logs in; the session, not the password, is
// remembered.
func (session *session) typeLogin() {
	email, confirmed := keyboard.Ask(keyboard.Text, "Email", session.profile.Email, maximumEmailLength)
	if !confirmed {
		return
	}
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
	session.remember(func(remembered *profile.Profile) {
		remembered.Email, remembered.Token, remembered.Username = email, logged.Token, logged.Username
	})
	session.restart("Logged in as " + logged.Username)
}

// logOut ends the session on the server and plays on as a guest.
func (session *session) logOut() {
	if server, err := session.api(); err == nil {
		// Leaving anyway: a session the server keeps a little longer does no harm.
		_ = server.LogOut(session.account)
	}
	session.forgetSession()
	session.restart("Logged out")
}

// forgetSession drops the session here and on the SD card; the email stays, to offer it again.
func (session *session) forgetSession() {
	session.account = account.Session{}
	session.remember(func(remembered *profile.Profile) { remembered.Token, remembered.Username = "", "" })
}

// keepLog writes status changes and, in the game, figures every few seconds.
func (session *session) keepLog() {
	session.log.keep(session.now)
	if session.status != session.logged {
		session.logged = session.status
		if session.status != "" {
			session.log.add(session.status)
		}
	}
	if session.screen == gameScreen && session.now-session.figuresAt >= figuresLogEvery {
		session.figuresAt = session.now
		shown := session.figures.Shown
		pending := 0
		if session.connection != nil {
			pending = session.connection.Pending()
		}
		session.log.add("figures: " + strconv.Itoa(shown.FramesPerSecond) + " fps, net " + strconv.Itoa(int(shown.NetworkMilliseconds)) +
			" ms, in " + strconv.Itoa(shown.BytesIn) + " B/s, out " + strconv.Itoa(shown.BytesOut) + " B/s, pending " + strconv.Itoa(pending) +
			" B, skipped " + strconv.Itoa(session.skipped) + ", players " + strconv.Itoa(shown.Players) +
			", music heard " + strconv.Itoa(int(session.sounds.musicSeconds())) + " s")
		session.skipped = 0
	}
}

// deathCaption says who destroyed us and when we fly again.
func (session *session) deathCaption() string {
	caption := ""
	if killer := session.game.KilledBy(); killer != "" {
		caption = "by " + killer + "  ·  "
	}
	if session.game.CanRespawn(session.now) {
		return caption + "A to fly again"
	}
	return caption + "back in " + strconv.Itoa(int(math.Ceil(session.game.RespawnIn(session.now)))) + " s"
}

// hear plays the sounds of what happened this frame: hits, blasts and the force field.
func (session *session) hear(events []game.Event) {
	for _, event := range events {
		switch event.Kind {
		case game.ShipHit:
			session.sounds.play(session.sounds.hit, 1)
		case game.ShipDestroyed:
			session.sounds.play(session.sounds.explosion, 1)
		}
	}
	if hit := session.game.EdgeHitAt; hit != 0 && hit != session.edgeHeard {
		session.edgeHeard = hit
		session.sounds.play(session.sounds.shield, 1)
	}
}

// applyVolumes gives the sound the settings' volumes.
func (session *session) applyVolumes() {
	settings := session.profile.Settings
	session.sounds.setVolumes(float64(settings.Effects), float64(settings.Music))
}
