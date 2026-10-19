package web

import (
	"errors"
	"io"
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

func dial(t *testing.T, server *httptest.Server) (pollingConnection, string) {
	address := strings.TrimPrefix(server.URL, "http://")
	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { connection.Close() })
	return pollingConnection{connection}, address
}

func TestALoginSetsItsCookie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		if request.Method != http.MethodPost || string(body) != `{"email":"a@b.c"}` {
			http.Error(writer, "unexpected", http.StatusBadRequest)
			return
		}
		http.SetCookie(writer, &http.Cookie{Name: "token", Value: "secret", HttpOnly: true, Secure: true})
		writer.Write([]byte(`{"success":true}`)) //nolint:errcheck // the test fails on its own if this is lost
	}))
	defer server.Close()
	connection, address := dial(t, server)
	response, err := Do(connection, Request{Method: http.MethodPost, Host: address, Path: "/login",
		Header: map[string]string{"Content-Type": "application/json"}, Body: []byte(`{"email":"a@b.c"}`)}, 2*time.Second)
	if err != nil || response.Status != http.StatusOK || string(response.Body) != `{"success":true}` {
		t.Fatalf("got %+v, %v", response, err)
	}
	if token, found := response.Cookie("token"); !found || token != "secret" {
		t.Fatalf("token %q, found %v", token, found)
	}
}

func TestALongAnswerInChunksIsJoined(t *testing.T) {
	long := strings.Repeat("ships ", 3000) // past net/http's buffer, so it is sent in chunks
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte(long)) //nolint:errcheck // as above
	}))
	defer server.Close()
	connection, address := dial(t, server)
	response, err := Do(connection, Request{Method: http.MethodGet, Host: address, Path: "/"}, 2*time.Second)
	if err != nil || string(response.Body) != long {
		t.Fatalf("got %d bytes, %v; encoding %v", len(response.Body), err, response.Header["transfer-encoding"])
	}
	if len(response.Header["transfer-encoding"]) == 0 {
		t.Fatal("the answer wasn't chunked: the test doesn't test what it says")
	}
}

// silentTransport never answers.
type silentTransport struct{}

func (silentTransport) Read([]byte) (int, error)        { return 0, nil }
func (silentTransport) Write(bytes []byte) (int, error) { return len(bytes), nil }

func TestASilentServerTimesOut(t *testing.T) {
	_, err := Do(silentTransport{}, Request{Method: http.MethodGet, Host: "h", Path: "/"}, 50*time.Millisecond)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("got %v, want %v", err, ErrTimeout)
	}
}

func TestHeadersCannotCarryNewLines(t *testing.T) {
	_, err := Do(silentTransport{}, Request{Method: http.MethodGet, Host: "h", Path: "/", Header: map[string]string{"Cookie": "a\r\nX: 1"}}, time.Second)
	if !errors.Is(err, ErrBadHeader) {
		t.Fatalf("got %v, want %v", err, ErrBadHeader)
	}
}

func TestEveryErrorHasItsOwnCode(t *testing.T) {
	codes := map[error]string{ErrTimeout: "web.timeout", ErrBadResponse: "web.bad-response", ErrBadHeader: "web.bad-header"}
	for err, code := range codes {
		if err.Error() != code {
			t.Errorf("%v has the code %q, want %q", err, err.Error(), code)
		}
	}
}
