// Package account logs a player in to the game's server and fetches the ships they may fly: the
// built-in ones and their own designs. It speaks the server's HTTP API (controllers/users.go and
// controllers/game.go), one request per connection, through package web.
package account

import (
	"encoding/json"
	"fmt"
	"time"

	"ships3d/client3ds/protocol"
	"ships3d/client3ds/web"
)

// Connection is a connection to the server for one request.
type Connection interface {
	web.Transport
	Close() error
}

// Dial opens a new connection to the server.
type Dial func() (Connection, error)

// Server is where the API is: how to reach it, and its host:port for the Host header.
type Server struct {
	Dial Dial
	Host string
}

// The few methods and statuses this package uses, as numbers and words of their own: net/http,
// which names them, isn't there for TinyGo on the console.
const (
	methodGet          = "GET"
	methodPost         = "POST"
	statusOK           = 200
	statusBadRequest   = 400
	statusUnauthorized = 401
)

// tokenCookie is the cookie the server keeps a session in (controllers/session.go).
const tokenCookie = "token"

// requestTimeout is how long a request may take before the player is told it failed.
const requestTimeout = 5 * time.Second

// Session is a logged-in player: the token to send as a cookie, and their name.
type Session struct {
	Token    string
	Username string
}

// Cookie is the header that carries the session, for the websocket's handshake.
func (session Session) Cookie() string { return tokenCookie + "=" + session.Token }

// LogIn asks the server for a session. A wrong email or password is ErrRefused.
func (server Server) LogIn(email, password string) (Session, error) {
	body, err := json.Marshal(map[string]any{"email": email, "password": password, "rememberMe": true})
	if err != nil {
		return Session{}, fmt.Errorf("%w: %w", ErrServer, err)
	}
	response, err := server.do(web.Request{Method: methodPost, Path: "/login",
		Header: map[string]string{"Content-Type": "application/json"}, Body: body})
	if err != nil {
		return Session{}, err
	}
	if response.Status == statusUnauthorized || response.Status == statusBadRequest {
		return Session{}, ErrRefused
	}
	token, found := response.Cookie(tokenCookie)
	if response.Status != statusOK || !found {
		return Session{}, fmt.Errorf("%w: status %d", ErrServer, response.Status)
	}
	var answer struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.Unmarshal(response.Body, &answer); err != nil {
		return Session{}, fmt.Errorf("%w: %w", ErrServer, err)
	}
	return Session{Token: token, Username: answer.User.Username}, nil
}

// Ships are the ships a player may fly: the built-in ones, and with a session their own designs.
type Ships struct {
	Defaults []protocol.Ship `json:"defaults"`
	Own      []protocol.Ship `json:"own"`
}

// Ships asks which ships the player may fly; session may be empty for a guest.
func (server Server) Ships(session Session) (Ships, error) {
	request := web.Request{Method: methodGet, Path: "/game/ships"}
	if session.Token != "" {
		request.Header = map[string]string{"Cookie": session.Cookie()}
	}
	response, err := server.do(request)
	if err != nil {
		return Ships{}, err
	}
	if response.Status != statusOK {
		return Ships{}, fmt.Errorf("%w: status %d", ErrServer, response.Status)
	}
	var ships Ships
	if err := json.Unmarshal(response.Body, &ships); err != nil {
		return Ships{}, fmt.Errorf("%w: %w", ErrServer, err)
	}
	return ships, nil
}

// LogOut ends the session on the server.
func (server Server) LogOut(session Session) error {
	response, err := server.do(web.Request{Method: methodPost, Path: "/logout", Header: map[string]string{"Cookie": session.Cookie()}})
	if err != nil {
		return err
	}
	if response.Status != statusOK {
		return fmt.Errorf("%w: status %d", ErrServer, response.Status)
	}
	return nil
}

func (server Server) do(request web.Request) (web.Response, error) {
	connection, err := server.Dial()
	if err != nil {
		return web.Response{}, err
	}
	defer connection.Close() //nolint:errcheck // one request per connection: nothing is lost if closing fails
	request.Host = server.Host
	return web.Do(connection, request, requestTimeout)
}
