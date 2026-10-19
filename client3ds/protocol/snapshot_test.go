package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"ships3d/client3ds/flight"
)

// serverSnapshot writes a snapshot as the server's encoder does.
func serverSnapshot(players int) string {
	type state struct {
		Id string          `json:"id"`
		P  flight.Vector   `json:"p"`
		Q  flight.Rotation `json:"q"`
		V  flight.Vector   `json:"v"`
	}
	message := struct {
		Type    string  `json:"type"`
		Players []state `json:"players"`
	}{Type: TypeSnapshot}
	for index := range players {
		value := float64(index)
		message.Players = append(message.Players, state{
			Id: fmt.Sprintf("player-%d", index),
			P:  flight.Vector{value * 1.5, -value, 1e-7 * value},
			Q:  flight.Rotation{0, 0.70710678, 0, 0.70710678},
			V:  flight.Vector{0, 0, -80},
		})
	}
	bytes, _ := json.Marshal(message)
	return string(bytes)
}

func TestTheReaderAgreesWithEncodingJson(t *testing.T) {
	text := serverSnapshot(30)
	var reader SnapshotReader
	got, err := reader.Read(text)
	if err != nil {
		t.Fatal(err)
	}
	var want snapshot
	if err := json.Unmarshal([]byte(text), &want); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want.Players) {
		t.Fatalf("%d players, want %d", len(got), len(want.Players))
	}
	for index := range got {
		if got[index] != want.Players[index] {
			t.Fatalf("player %d: got %+v, want %+v", index, got[index], want.Players[index])
		}
	}
}

func TestReadingASnapshotAllocatesNothingOnceWarm(t *testing.T) {
	text := serverSnapshot(60)
	var reader SnapshotReader
	if _, err := reader.Read(text); err != nil {
		t.Fatal(err)
	}
	allocations := testing.AllocsPerRun(50, func() {
		if _, err := reader.Read(text); err != nil {
			t.Fatal(err)
		}
	})
	if allocations > 0 {
		t.Fatalf("%v allocations a snapshot", allocations)
	}
}

func TestUnknownFieldsAreSkipped(t *testing.T) {
	var reader SnapshotReader
	text := `{"type":"snapshot","tick":12,"players":[{"id":"a","extra":{"x":[1,true,null]},"p":[1,2,3],"q":[0,0,0,1],"v":[0,0,0]}]}`
	got, err := reader.Read(text)
	if err != nil || len(got) != 1 || got[0].Position != (flight.Vector{1, 2, 3}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestBrokenSnapshotsAreRefusedWithTheirCode(t *testing.T) {
	cases := []string{
		`{"type":"snapshot","players":[{"id":"a","p":[1,2],"q":[0,0,0,1],"v":[0,0,0]}]}`,
		`{"type":"snapshot","players":[{"id":"a","p":[1,2,3,4]`,
		`{"type":"snapshot","players":[{"id":"a\"b"}]}`,
		strings.Repeat("{", 3),
	}
	for _, text := range cases {
		var reader SnapshotReader
		if _, err := reader.Read(text); !errors.Is(err, ErrUnreadable) {
			t.Errorf("%s: got %v, want %v", text, err, ErrUnreadable)
		}
	}
}
