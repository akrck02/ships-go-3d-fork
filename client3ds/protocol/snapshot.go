package protocol

import (
	"fmt"
	"strconv"
)

// SnapshotReader reads snapshots by hand. They are most of what arrives, twenty a second with
// every moved player in each, and encoding/json reads them through reflection, which on TinyGo
// took most of a frame with sixty players. This reader knows their one shape:
//
//	{"type":"snapshot","players":[{"id":"…","p":[x,y,z],"q":[x,y,z,w],"v":[x,y,z]},…]}
//
// and keeps its list and the players' ids between calls, so reading allocates nothing once warm.
type SnapshotReader struct {
	states []PlayerState
	ids    map[string]string // each id read once, then reused
	text   string
	at     int
}

// Read returns the moved players of a snapshot. The slice is reused by the next call.
func (reader *SnapshotReader) Read(text string) ([]PlayerState, error) {
	if reader.ids == nil {
		reader.ids = map[string]string{}
	}
	reader.text, reader.at, reader.states = text, 0, reader.states[:0]
	if err := reader.readObject(reader.readTopField); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	return reader.states, nil
}

// errSyntax is what the reader found where it didn't expect it.
type errSyntax struct {
	at       int
	expected string
}

func (err errSyntax) Error() string {
	return "expected " + err.expected + " at " + strconv.Itoa(err.at)
}

func (reader *SnapshotReader) readTopField(name string) error {
	switch name {
	case "type":
		_, err := reader.readString()
		return err
	case "players":
		return reader.readArray(func() error {
			reader.states = append(reader.states, PlayerState{})
			return reader.readObject(reader.readPlayerField)
		})
	}
	return reader.skipValue()
}

func (reader *SnapshotReader) readPlayerField(name string) error {
	state := &reader.states[len(reader.states)-1]
	switch name {
	case "id":
		id, err := reader.readString()
		state.Id = reader.intern(id)
		return err
	case "p":
		return reader.readNumbers(state.Position[:])
	case "q":
		return reader.readNumbers(state.Rotation[:])
	case "v":
		return reader.readNumbers(state.Velocity[:])
	}
	return reader.skipValue()
}

// intern returns the id kept from an earlier snapshot, so each player's id is one string.
func (reader *SnapshotReader) intern(id string) string {
	if kept, found := reader.ids[id]; found {
		return kept
	}
	reader.ids[id] = id
	return id
}

func (reader *SnapshotReader) skipSpace() {
	for reader.at < len(reader.text) {
		switch reader.text[reader.at] {
		case ' ', '\t', '\n', '\r':
			reader.at++
		default:
			return
		}
	}
}

// expect reads one character, after any space.
func (reader *SnapshotReader) expect(character byte) error {
	reader.skipSpace()
	if reader.at >= len(reader.text) || reader.text[reader.at] != character {
		return errSyntax{reader.at, string(character)}
	}
	reader.at++
	return nil
}

// peek is the next character after any space, or 0 at the end.
func (reader *SnapshotReader) peek() byte {
	reader.skipSpace()
	if reader.at >= len(reader.text) {
		return 0
	}
	return reader.text[reader.at]
}

// readObject reads {"name": value, …}, handing each name to field, which reads the value.
func (reader *SnapshotReader) readObject(field func(name string) error) error {
	if err := reader.expect('{'); err != nil {
		return err
	}
	if reader.peek() == '}' {
		reader.at++
		return nil
	}
	for {
		name, err := reader.readString()
		if err != nil {
			return err
		}
		if err := reader.expect(':'); err != nil {
			return err
		}
		if err := field(name); err != nil {
			return err
		}
		if reader.peek() == ',' {
			reader.at++
			continue
		}
		return reader.expect('}')
	}
}

// readArray reads [value, …], with item reading each value.
func (reader *SnapshotReader) readArray(item func() error) error {
	if err := reader.expect('['); err != nil {
		return err
	}
	if reader.peek() == ']' {
		reader.at++
		return nil
	}
	for {
		if err := item(); err != nil {
			return err
		}
		if reader.peek() == ',' {
			reader.at++
			continue
		}
		return reader.expect(']')
	}
}

// readString reads a string with no escapes, which is all the server writes in a snapshot (ids
// and the type); one with escapes is refused rather than misread.
func (reader *SnapshotReader) readString() (string, error) {
	if err := reader.expect('"'); err != nil {
		return "", err
	}
	start := reader.at
	for reader.at < len(reader.text) {
		switch reader.text[reader.at] {
		case '"':
			reader.at++
			return reader.text[start : reader.at-1], nil
		case '\\':
			return "", errSyntax{reader.at, "a string without escapes"}
		}
		reader.at++
	}
	return "", errSyntax{reader.at, `"`}
}

// readNumbers reads an array of exactly len(into) numbers.
func (reader *SnapshotReader) readNumbers(into []float64) error {
	count := 0
	err := reader.readArray(func() error {
		if count == len(into) {
			return errSyntax{reader.at, "]"}
		}
		value, err := reader.readNumber()
		into[count] = value
		count++
		return err
	})
	if err == nil && count != len(into) {
		return errSyntax{reader.at, strconv.Itoa(len(into)) + " numbers"}
	}
	return err
}

func (reader *SnapshotReader) readNumber() (float64, error) {
	reader.skipSpace()
	start := reader.at
	for reader.at < len(reader.text) && isNumberCharacter(reader.text[reader.at]) {
		reader.at++
	}
	value, err := strconv.ParseFloat(reader.text[start:reader.at], 64)
	if err != nil {
		return 0, errSyntax{start, "a number"}
	}
	return value, nil
}

func isNumberCharacter(character byte) bool {
	return character >= '0' && character <= '9' || character == '-' || character == '+' || character == '.' || character == 'e' || character == 'E'
}

// skipValue passes over a value of a field the reader doesn't use, so a server that adds one to
// snapshots doesn't break the client.
func (reader *SnapshotReader) skipValue() error {
	switch reader.peek() {
	case '"':
		_, err := reader.readString()
		return err
	case '{':
		return reader.readObject(func(string) error { return reader.skipValue() })
	case '[':
		return reader.readArray(reader.skipValue)
	case 't', 'f', 'n':
		for reader.at < len(reader.text) && reader.text[reader.at] >= 'a' && reader.text[reader.at] <= 'z' {
			reader.at++
		}
		return nil
	}
	_, err := reader.readNumber()
	return err
}
