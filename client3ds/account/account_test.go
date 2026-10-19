package account

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// pollingConnection makes a net.Conn read like the console's: at once, 0 bytes when nothing came.
type pollingConnection struct{ net.Conn }

func (connection pollingConnection) Read(buffer []byte) (int, error) {
	connection.SetReadDeadline(time.Now().Add(time.Millisecond)) //nolint:errcheck // a failed deadline only makes the read wait
	count, err := connection.Conn.Read(buffer)
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		return count, nil
	}
	return count, err
}

// fakeServer answers as the game's server does, for one user with one ship of their own.
func fakeServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", func(writer http.ResponseWriter, request *http.Request) {
		var body struct{ Email, Password string }
		json.NewDecoder(request.Body).Decode(&body) //nolint:errcheck // a bad body fails the check below
		if body.Email != "pilot@example.com" || body.Password != "right" {
			http.Error(writer, `{"success":false}`, http.StatusUnauthorized)
			return
		}
		http.SetCookie(writer, &http.Cookie{Name: "token", Value: "abc", HttpOnly: true, Secure: true})
		writer.Write([]byte(`{"success":true,"user":{"username":"pilot"}}`)) //nolint:errcheck // the test fails on its own
	})
	mux.HandleFunc("GET /game/ships", func(writer http.ResponseWriter, request *http.Request) {
		own := `[]`
		if cookie, err := request.Cookie("token"); err == nil && cookie.Value == "abc" {
			own = `[{"id":"p1","name":"Mine","layers":[]}]`
		}
		writer.Write([]byte(`{"defaults":[{"id":"default:arrow","name":"Arrow","layers":[]}],"own":` + own + `}`)) //nolint:errcheck // as above
	})
	mux.HandleFunc("GET /userInfo", func(writer http.ResponseWriter, request *http.Request) {
		if cookie, err := request.Cookie("token"); err == nil && cookie.Value == "abc" {
			writer.Write([]byte(`{"user":{"username":"pilot"}}`)) //nolint:errcheck // the test fails on its own
			return
		}
		writer.Write([]byte(`{"user":null}`)) //nolint:errcheck // as above
	})
	mux.HandleFunc("POST /logout", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte(`{"success":true}`)) //nolint:errcheck // as above
	})
	return httptest.NewServer(mux)
}

func serverFor(t *testing.T, test *httptest.Server) Server {
	address := strings.TrimPrefix(test.URL, "http://")
	return Server{Host: address, Dial: func() (Connection, error) {
		connection, err := net.Dial("tcp", address)
		if err != nil {
			return nil, err
		}
		return pollingConnection{connection}, nil
	}}
}

func TestAPlayerLogsInAndSeesTheirShips(t *testing.T) {
	test := fakeServer()
	defer test.Close()
	server := serverFor(t, test)
	session, err := server.LogIn("pilot@example.com", "right")
	if err != nil || session.Token != "abc" || session.Username != "pilot" {
		t.Fatalf("got %+v, %v", session, err)
	}
	ships, err := server.Ships(session)
	if err != nil || len(ships.Defaults) != 1 || len(ships.Own) != 1 || ships.Own[0].Id != "p1" {
		t.Fatalf("got %+v, %v", ships, err)
	}
	if err := server.LogOut(session); err != nil {
		t.Fatal(err)
	}
}

func TestAGuestSeesOnlyTheBuiltInShips(t *testing.T) {
	test := fakeServer()
	defer test.Close()
	ships, err := serverFor(t, test).Ships(Session{})
	if err != nil || len(ships.Own) != 0 || len(ships.Defaults) != 1 {
		t.Fatalf("got %+v, %v", ships, err)
	}
}

func TestAWrongPasswordIsRefused(t *testing.T) {
	test := fakeServer()
	defer test.Close()
	if _, err := serverFor(t, test).LogIn("pilot@example.com", "wrong"); !errors.Is(err, ErrRefused) {
		t.Fatalf("got %v, want %v", err, ErrRefused)
	}
}

func TestEveryErrorHasItsOwnCode(t *testing.T) {
	codes := map[error]string{ErrRefused: "account.refused", ErrServer: "account.server-error", ErrAddress: "account.bad-address"}
	for err, code := range codes {
		if err.Error() != code {
			t.Errorf("%v has the code %q, want %q", err, err.Error(), code)
		}
	}
}

func TestAddressesAreUnderstood(t *testing.T) {
	cases := map[string]Address{
		"192.168.1.20:3000":           {Host: "192.168.1.20", Port: 3000},
		"192.168.1.20":                {Host: "192.168.1.20", Port: DefaultPort},
		"https://example.org:3000":    {Host: "example.org", Port: 3000, Secure: true},
		" HTTPS://example.org:3000/ ": {Host: "example.org", Port: 3000, Secure: true},
		"wss://example.org":           {Host: "example.org", Port: DefaultPort, Secure: true},
	}
	for typed, want := range cases {
		got, err := ParseAddress(typed)
		if err != nil || got != want {
			t.Errorf("%q: got %+v, %v; want %+v", typed, got, err, want)
		}
	}
	for _, typed := range []string{"", "https://", "example.org:port", "example.org:70000"} {
		if _, err := ParseAddress(typed); !errors.Is(err, ErrAddress) {
			t.Errorf("%q: got %v, want %v", typed, err, ErrAddress)
		}
	}
}

func TestARememberedSessionIsCheckedWithTheServer(t *testing.T) {
	test := fakeServer()
	defer test.Close()
	server := serverFor(t, test)
	if name, valid, err := server.Who(Session{Token: "abc"}); err != nil || !valid || name != "pilot" {
		t.Fatalf("got %q %v %v", name, valid, err)
	}
	if _, valid, err := server.Who(Session{Token: "expired"}); err != nil || valid {
		t.Fatalf("an unknown session came back valid (%v)", err)
	}
}
