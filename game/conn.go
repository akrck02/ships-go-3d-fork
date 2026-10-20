package game

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"ships3d/i18n"
)

// serveConnection upgrades an HTTP request and reads the connection until it
// closes, handing each message to handle with the connection's id. leave is
// called once it has closed. Both game modes run their sockets through it, so
// the limits (frame size, flood protection, pings) are the same everywhere.
func serveConnection(
	upgrader *websocket.Upgrader,
	now func() time.Time,
	w http.ResponseWriter,
	r *http.Request,
	handle func(id string, c *client, data []byte),
	leave func(id string),
) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := newClient(conn)
	c.lang = i18n.FromRequest(r)
	go c.writePump()

	id := uuid.NewString()
	defer func() {
		c.close()
		leave(id)
	}()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongTimeout))
	})

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
		if !c.allowMessage(now()) {
			log.Printf("game: disconnecting %s, too many messages (sustained over %.0f/s)", id, messageRate)
			// Say why, rather than just dropping the connection.
			sendTo(c, errorMsg{Type: msgError, Message: i18n.New("tooManyMessages").In(c.lang)})
			time.Sleep(rateLimitGrace)
			return
		}
		handle(id, c, data)
	}
}

func sendTo(c *client, msg any) {
	frame, err := json.Marshal(msg)
	if err != nil {
		log.Println("game: marshal:", err)
		return
	}
	c.enqueue(frame)
}
