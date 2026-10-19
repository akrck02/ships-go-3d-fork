// Package web makes plain HTTP/1.1 requests over a connection whose reads return at once, as the
// console's are: enough for a game to log in and fetch a list. One request per connection: it
// asks the server to close the connection after answering, and reads until it does.
package web

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Transport is the connection under the request: its Read returns at once, with 0 bytes and no
// error when nothing has arrived, and an error once the other end has closed.
type Transport interface {
	Read(buffer []byte) (int, error)
	Write(bytes []byte) (int, error)
}

// Request is what to ask for.
type Request struct {
	Method, Host, Path string
	Header             map[string]string
	Body               []byte
}

// Response is what came back. Header names are in lower case.
type Response struct {
	Status int
	Header map[string][]string
	Body   []byte
}

// Cookie is the value the response sets for a cookie, and whether it sets one.
func (response Response) Cookie(name string) (string, bool) {
	for _, line := range response.Header["set-cookie"] {
		pair, _, _ := strings.Cut(line, ";")
		key, value, found := strings.Cut(pair, "=")
		if found && strings.TrimSpace(key) == name {
			return strings.TrimSpace(value), true
		}
	}
	return "", false
}

// How long Do waits between reads, and how much one read asks for.
const (
	pollInterval = 5 * time.Millisecond
	chunkBytes   = 4096
)

// Do sends the request and waits for the whole answer, at most timeout. It blocks the program
// while it waits: meant for an action the player asked for, like logging in, not for every frame.
func Do(transport Transport, request Request, timeout time.Duration) (Response, error) {
	if err := write(transport, request); err != nil {
		return Response{}, err
	}
	raw, err := readUntilClosed(transport, timeout)
	if err != nil {
		return Response{}, err
	}
	return parse(raw)
}

func write(transport Transport, request Request) error {
	var text strings.Builder
	fmt.Fprintf(&text, "%s %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n", request.Method, request.Path, request.Host)
	for name, value := range request.Header {
		if strings.ContainsAny(name, "\r\n:") || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%w: %q", ErrBadHeader, name)
		}
		fmt.Fprintf(&text, "%s: %s\r\n", name, value)
	}
	if request.Body != nil {
		fmt.Fprintf(&text, "Content-Length: %d\r\n", len(request.Body))
	}
	text.WriteString("\r\n")
	if _, err := transport.Write(append([]byte(text.String()), request.Body...)); err != nil {
		return err
	}
	return nil
}

// readUntilClosed reads everything until the server closes the connection.
func readUntilClosed(transport Transport, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	received := []byte{}
	chunk := make([]byte, chunkBytes)
	for {
		count, err := transport.Read(chunk)
		received = append(received, chunk[:count]...)
		if err != nil {
			return received, nil // the server closed the connection: the answer is whole
		}
		if count > 0 {
			continue
		}
		if time.Now().After(deadline) {
			return nil, ErrTimeout
		}
		time.Sleep(pollInterval)
	}
}

func parse(raw []byte) (Response, error) {
	end := bytes.Index(raw, []byte("\r\n\r\n"))
	if end < 0 {
		return Response{}, fmt.Errorf("%w: no end to the header", ErrBadResponse)
	}
	lines := strings.Split(string(raw[:end]), "\r\n")
	fields := strings.Fields(lines[0])
	if len(fields) < 2 {
		return Response{}, fmt.Errorf("%w: %q", ErrBadResponse, lines[0])
	}
	status, err := strconv.Atoi(fields[1])
	if err != nil {
		return Response{}, fmt.Errorf("%w: %q", ErrBadResponse, lines[0])
	}
	response := Response{Status: status, Header: map[string][]string{}}
	for _, line := range lines[1:] {
		name, value, found := strings.Cut(line, ":")
		if found {
			key := strings.ToLower(strings.TrimSpace(name))
			response.Header[key] = append(response.Header[key], strings.TrimSpace(value))
		}
	}
	body := raw[end+4:]
	if encoding := response.Header["transfer-encoding"]; len(encoding) > 0 && strings.EqualFold(encoding[0], "chunked") {
		body, err = unchunk(body)
		if err != nil {
			return Response{}, err
		}
	}
	response.Body = body
	return response, nil
}

// unchunk joins a body sent in chunks: each a length in hexadecimal, its bytes, and a 0 at the end.
func unchunk(body []byte) ([]byte, error) {
	joined := []byte{}
	for {
		end := bytes.Index(body, []byte("\r\n"))
		if end < 0 {
			return nil, fmt.Errorf("%w: a chunk without its length", ErrBadResponse)
		}
		sizeText, _, _ := strings.Cut(string(body[:end]), ";")
		size, err := strconv.ParseInt(strings.TrimSpace(sizeText), 16, 64)
		if err != nil || size < 0 {
			return nil, fmt.Errorf("%w: chunk length %q", ErrBadResponse, sizeText)
		}
		body = body[end+2:]
		if size == 0 {
			return joined, nil
		}
		if int64(len(body)) < size {
			return nil, fmt.Errorf("%w: a chunk cut short", ErrBadResponse)
		}
		joined = append(joined, body[:size]...)
		body = bytes.TrimPrefix(body[size:], []byte("\r\n"))
	}
}
