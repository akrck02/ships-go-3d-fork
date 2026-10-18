package websocket

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gorilla "github.com/gorilla/websocket"
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

// echoServer answers every text message with the same text, and says "hello" first.
func echoServer() *httptest.Server {
	upgrader := gorilla.Upgrader{}
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			return // a test that closed its connection early
		}
		defer connection.Close()
		connection.WriteMessage(gorilla.TextMessage, []byte("hello")) //nolint:errcheck // the test fails on its own if this is lost
		for {
			kind, message, err := connection.ReadMessage()
			if err != nil {
				return
			}
			connection.WriteMessage(kind, message) //nolint:errcheck // as above
		}
	}))
}

func connect(t *testing.T, server *httptest.Server) *Client {
	address := strings.TrimPrefix(server.URL, "http://")
	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { connection.Close() })
	client, err := Connect(pollingConnection{connection}, address, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// receiveUntil polls like a game loop until count messages have arrived.
func receiveUntil(t *testing.T, client *Client, count int) []string {
	received := []string{}
	for attempt := 0; attempt < 2000 && len(received) < count; attempt++ {
		messages, err := client.Receive()
		if err != nil {
			t.Fatal(err)
		}
		received = append(received, messages...)
	}
	return received
}

func TestAClientTalksToAGorillaServer(t *testing.T) {
	server := echoServer()
	defer server.Close()
	client := connect(t, server)
	if got := receiveUntil(t, client, 1); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("got %q before sending, want the greeting", got)
	}
	long := strings.Repeat("ships ", 30000) // over 65535 bytes: the 64-bit length
	for _, text := range []string{"short", strings.Repeat("x", 300), long} {
		if err := client.Send(text); err != nil {
			t.Fatal(err)
		}
		if got := receiveUntil(t, client, 1); len(got) != 1 || got[0] != text {
			t.Fatalf("a message of %d bytes came back changed", len(text))
		}
	}
}

func TestNothingIsSentBeforeTheHandshake(t *testing.T) {
	server := echoServer()
	defer server.Close()
	client := connect(t, server)
	if err := client.Send("early"); !errors.Is(err, ErrNotOpen) {
		t.Fatalf("got %v, want %v", err, ErrNotOpen)
	}
}

// recordingTransport keeps what is written and never receives anything.
type recordingTransport struct{ written []byte }

func (transport *recordingTransport) Read([]byte) (int, error) { return 0, nil }
func (transport *recordingTransport) Write(bytes []byte) (int, error) {
	transport.written = append(transport.written, bytes...)
	return len(bytes), nil
}

func TestClientFramesAreMasked(t *testing.T) {
	transport := &recordingTransport{}
	client := &Client{transport: transport, open: true}
	if err := client.Send("abc"); err != nil {
		t.Fatal(err)
	}
	frame := transport.written
	if frame[0] != finalBit|opcodeText || frame[1] != maskBit|3 {
		t.Fatalf("header %x, want a final masked text frame of 3 bytes", frame[:2])
	}
	key := frame[2:6]
	for index, want := range []byte("abc") {
		if frame[6+index]^key[index] != want {
			t.Fatalf("byte %d doesn't unmask to %q", index, want)
		}
	}
}

func TestAPingIsAnsweredWithAPong(t *testing.T) {
	transport := &recordingTransport{}
	client := &Client{transport: transport, open: true}
	client.pending = []byte{finalBit | opcodePing, 2, 'h', 'i'}
	if _, err := client.Receive(); err != nil {
		t.Fatal(err)
	}
	if len(transport.written) == 0 || transport.written[0] != finalBit|opcodePong {
		t.Fatalf("wrote %x, want a pong", transport.written)
	}
}

func TestAWrongAcceptIsRefused(t *testing.T) {
	client := &Client{transport: &recordingTransport{}, key: "dGhlIHNhbXBsZSBub25jZQ=="}
	client.pending = []byte("HTTP/1.1 101 Switching Protocols\r\nSec-WebSocket-Accept: nope\r\n\r\n")
	if _, err := client.Receive(); !errors.Is(err, ErrHandshake) {
		t.Fatalf("got %v, want %v", err, ErrHandshake)
	}
}

func TestTheAcceptMatchesTheRfcExample(t *testing.T) {
	if got := acceptFor("dGhlIHNhbXBsZSBub25jZQ=="); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("got %s", got)
	}
}

func TestEveryErrorHasItsOwnCode(t *testing.T) {
	codes := map[error]string{
		ErrHandshake: "websocket.handshake-refused", ErrNotOpen: "websocket.not-open",
		ErrBadFrame: "websocket.bad-frame", ErrClosed: "websocket.closed",
		ErrTooLarge: "websocket.message-too-large", ErrUnsupported: "websocket.unsupported-message",
	}
	for err, code := range codes {
		if err.Error() != code {
			t.Errorf("%v has the code %q, want %q", err, err.Error(), code)
		}
	}
}
