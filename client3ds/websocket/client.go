// Package websocket is the client side of the websocket protocol (RFC 6455), small enough for a
// game console: text messages only, over any connection whose reads return at once.
//
// It never waits for the network. A program calls Receive once a frame and gets the messages that
// have arrived whole since the last call; Send writes one message.
package websocket

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"strings"
)

// Transport is the connection under the websocket: a TCP connection whose Read returns at once,
// with 0 bytes and no error when nothing has arrived.
type Transport interface {
	Read(buffer []byte) (int, error)
	Write(bytes []byte) (int, error)
}

// MaximumMessageBytes bounds one message, so a broken server can't make the client take all its
// memory. The largest message is the welcome, which carries every player's ship: a design at the
// server's limit of 400 parts is about 60 KB, so sixty such players fit.
const MaximumMessageBytes = 8 << 20

// acceptSalt is the fixed text RFC 6455 adds to the key to prove the server speaks websocket.
const acceptSalt = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// The frame types the protocol defines (RFC 6455, section 5.2).
const (
	opcodeContinuation = 0x0
	opcodeText         = 0x1
	opcodeBinary       = 0x2
	opcodeClose        = 0x8
	opcodePing         = 0x9
	opcodePong         = 0xA
)

// maximumHeaderBytes is the longest frame header: two bytes, a 64-bit length and a mask key.
const maximumHeaderBytes = 14

const (
	finalBit       = 0x80
	maskBit        = 0x80
	opcodeBits     = 0x0F
	lengthBits     = 0x7F
	length16Marker = 126
	length64Marker = 127
	maskKeyBytes   = 4
)

// readChunkBytes is how much one Read asks the transport for.
const readChunkBytes = 4096

// Client is one websocket connection, from the client's side.
type Client struct {
	transport Transport
	key       string
	open      bool
	closed    bool
	pending   []byte // received and not yet understood
	message   []byte // the pieces of a message split over several frames
	inMessage bool   // a message's first frame came and its last hasn't
	chunk     []byte
	frame     []byte // the frame being sent, reused
	messages  []string
}

// Connect starts the handshake: it asks the server at host for path, adding header's lines
// (such as a Cookie). The connection opens once Receive has read the server's answer.
func Connect(transport Transport, host, path string, header map[string]string) (*Client, error) {
	var nonce [16]byte
	for index := range nonce {
		// The key only has to differ between connections, to stop caches answering for the
		// server; it guards nothing, so a fast generator is enough.
		nonce[index] = byte(rand.Uint32())
	}
	for name, value := range header {
		if strings.ContainsAny(name, "\r\n:") || strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("%w: %q", ErrBadHeader, name)
		}
	}
	client := &Client{transport: transport, key: base64.StdEncoding.EncodeToString(nonce[:]), chunk: make([]byte, readChunkBytes)}
	var request strings.Builder
	fmt.Fprintf(&request, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n", path, host)
	fmt.Fprintf(&request, "Sec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n", client.key)
	for name, value := range header {
		fmt.Fprintf(&request, "%s: %s\r\n", name, value)
	}
	request.WriteString("\r\n")
	if _, err := transport.Write([]byte(request.String())); err != nil {
		return nil, err
	}
	return client, nil
}

// Open tells whether the handshake is done and messages can be sent.
func (client *Client) Open() bool { return client.open && !client.closed }

// Receive reads what has arrived and returns the text messages now complete. The slice is reused
// by the next call. It returns ErrClosed once the server has closed the connection.
func (client *Client) Receive() ([]string, error) {
	client.messages = client.messages[:0]
	if client.closed {
		return nil, ErrClosed
	}
	if err := client.readAvailable(); err != nil {
		client.closed = true
		return client.messages, err
	}
	if !client.open {
		done, err := client.readHandshake()
		if err != nil {
			client.closed = true
			return nil, err
		}
		if !done {
			return nil, nil
		}
	}
	if err := client.readFrames(); err != nil {
		client.closed = true
		return client.messages, err
	}
	return client.messages, nil
}

// readAvailable adds to pending what the transport has, without waiting. A server sending
// faster than we read can't make pending grow past one whole message.
func (client *Client) readAvailable() error {
	for len(client.pending) <= MaximumMessageBytes+maximumHeaderBytes {
		count, err := client.transport.Read(client.chunk)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrClosed, err)
		}
		if count == 0 {
			return nil
		}
		client.pending = append(client.pending, client.chunk[:count]...)
	}
	return nil
}

// readFrames takes the whole frames in pending as messages, and keeps the rest for later.
func (client *Client) readFrames() error {
	read := 0
	for {
		consumed, err := client.readFrame(client.pending[read:])
		if err != nil {
			return err
		}
		if consumed == 0 {
			break
		}
		read += consumed
	}
	// Keep what is left at the start of the same buffer, so it is reused frame after frame.
	client.pending = client.pending[:copy(client.pending, client.pending[read:])]
	return nil
}

// readHandshake checks the server's answer once it has arrived whole.
func (client *Client) readHandshake() (bool, error) {
	end := bytes.Index(client.pending, []byte("\r\n\r\n"))
	if end < 0 {
		return false, nil
	}
	answer := string(client.pending[:end])
	client.pending = client.pending[end+4:]
	lines := strings.Split(answer, "\r\n")
	if !strings.Contains(lines[0], " 101 ") {
		return false, fmt.Errorf("%w: %s", ErrHandshake, lines[0])
	}
	expected := acceptFor(client.key)
	for _, line := range lines[1:] {
		name, value, found := strings.Cut(line, ":")
		if found && strings.EqualFold(strings.TrimSpace(name), "Sec-WebSocket-Accept") && strings.TrimSpace(value) == expected {
			client.open = true
			return true, nil
		}
	}
	return false, fmt.Errorf("%w: wrong Sec-WebSocket-Accept", ErrHandshake)
}

// acceptFor is the answer a real websocket server gives to a key.
func acceptFor(key string) string {
	sum := sha1.Sum([]byte(key + acceptSalt))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// readFrame understands the first frame of pending, if it has arrived whole, and returns how many
// bytes it took (0 when it hasn't arrived whole yet).
func (client *Client) readFrame(pending []byte) (int, error) {
	if len(pending) < 2 {
		return 0, nil
	}
	final := pending[0]&finalBit != 0
	opcode := pending[0] & opcodeBits
	masked := pending[1]&maskBit != 0
	length := uint64(pending[1] & lengthBits)
	header := 2
	switch length {
	case length16Marker:
		if len(pending) < header+2 {
			return 0, nil
		}
		length = uint64(binary.BigEndian.Uint16(pending[header:]))
		header += 2
	case length64Marker:
		if len(pending) < header+8 {
			return 0, nil
		}
		length = binary.BigEndian.Uint64(pending[header:])
		header += 8
	}
	if masked {
		return 0, fmt.Errorf("%w: the server masked a frame", ErrBadFrame)
	}
	if length > MaximumMessageBytes || uint64(len(client.message))+length > MaximumMessageBytes {
		return 0, ErrTooLarge
	}
	total := header + int(length)
	if len(pending) < total {
		return 0, nil
	}
	return total, client.handleFrame(opcode, final, pending[header:total])
}

// handleFrame acts on one whole frame.
func (client *Client) handleFrame(opcode byte, final bool, payload []byte) error {
	switch opcode {
	case opcodeText, opcodeContinuation:
		// A message starts with a text frame and goes on with continuations (RFC 6455, 5.4).
		if (opcode == opcodeText) == client.inMessage {
			return fmt.Errorf("%w: a message's frames out of order", ErrBadFrame)
		}
		client.message = append(client.message, payload...)
		client.inMessage = !final
		if final {
			client.messages = append(client.messages, string(client.message))
			client.message = client.message[:0]
		}
		return nil
	case opcodePing:
		return client.writeFrame(opcodePong, payload)
	case opcodePong:
		return nil
	case opcodeClose:
		// Answering the close is a courtesy: the connection ends either way.
		_ = client.writeFrame(opcodeClose, nil)
		return ErrClosed
	case opcodeBinary:
		return fmt.Errorf("%w: binary", ErrUnsupported)
	}
	return fmt.Errorf("%w: opcode %d", ErrBadFrame, opcode)
}

// Send sends one text message.
func (client *Client) Send(text string) error {
	if !client.Open() {
		return ErrNotOpen
	}
	return client.writeFrame(opcodeText, []byte(text))
}

// Close tells the server the client is leaving.
func (client *Client) Close() error {
	if !client.Open() {
		return nil
	}
	client.closed = true
	return client.writeFrame(opcodeClose, nil)
}

// writeFrame sends one frame, masked as every client frame must be.
func (client *Client) writeFrame(opcode byte, payload []byte) error {
	frame := append(client.frame[:0], finalBit|opcode)
	switch {
	case len(payload) < length16Marker:
		frame = append(frame, maskBit|byte(len(payload)))
	case len(payload) <= 0xFFFF:
		frame = append(frame, maskBit|length16Marker)
		frame = binary.BigEndian.AppendUint16(frame, uint16(len(payload)))
	default:
		frame = append(frame, maskBit|length64Marker)
		frame = binary.BigEndian.AppendUint64(frame, uint64(len(payload)))
	}
	mask := rand.Uint32()
	frame = binary.BigEndian.AppendUint32(frame, mask)
	key := frame[len(frame)-maskKeyBytes:]
	start := len(frame)
	frame = append(frame, payload...)
	for index := range payload {
		frame[start+index] ^= key[index%maskKeyBytes]
	}
	client.frame = frame
	_, err := client.transport.Write(frame)
	return err
}
