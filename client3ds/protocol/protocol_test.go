package protocol

import (
	"encoding/json"
	"errors"
	"testing"

	"ships3d/client3ds/flight"
)

func TestAWelcomeIsRead(t *testing.T) {
	text := `{"type":"welcome","id":"a","settings":{"tickRate":20,"shipRadius":4,"maxSpeed":80},
	"players":[{"id":"a","name":"Guest","ship":{"id":"default:arrow","name":"Arrow","layers":[{"name":"Hull","visible":true,
	"parts":[{"type":"box","position":[0,0,0],"rotation":[0,0,0],"scale":[1,2,3],"color":"#ffffff","mirror":true}]}]},
	"scale":0.5,"life":10,"p":[1,2,3],"q":[0,0,0,1]}]}`
	message, _, err := Read(text)
	if err != nil {
		t.Fatal(err)
	}
	if message.Settings.ShipRadius != 4 || message.Settings.MaxSpeed != 80 {
		t.Fatalf("settings %+v", message.Settings)
	}
	part := message.Players[0].Ship.Layers[0].Parts[0]
	if part.Scale != (flight.Vector{1, 2, 3}) || !part.Mirror || message.Players[0].Position != (flight.Vector{1, 2, 3}) {
		t.Fatalf("player %+v", message.Players[0])
	}
}

func TestASnapshotGivesItsMovedPlayers(t *testing.T) {
	message, states, err := Read(`{"type":"snapshot","players":[{"id":"b","p":[1,0,0],"q":[0,0,0,1],"v":[0,0,5]}]}`)
	if err != nil || message.Type != TypeSnapshot || len(states) != 1 || states[0].Velocity[2] != 5 {
		t.Fatalf("got %+v %+v %v", message, states, err)
	}
}

func TestOutgoingMessagesUseTheServersFieldNames(t *testing.T) {
	cases := map[string][]string{
		Join("default:arrow", "3DS"):                               {"type", "shipId", "name"},
		State(flight.Vector{}, flight.NoRotation, flight.Vector{}): {"type", "p", "q", "v"},
		Fire("1", flight.Vector{}, flight.Forward):                 {"type", "id", "p", "d"},
		Hit("1", "b"): {"type", "bulletId", "target"},
		Respawn():     {"type"},
	}
	for text, fields := range cases {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(text), &decoded); err != nil {
			t.Fatal(err)
		}
		for _, field := range fields {
			if _, found := decoded[field]; !found {
				t.Errorf("%s has no %q", text, field)
			}
		}
	}
}

func TestBrokenJsonHasItsCode(t *testing.T) {
	if _, _, err := Read("{not json"); !errors.Is(err, ErrUnreadable) {
		t.Fatalf("got %v, want %v", err, ErrUnreadable)
	}
	if ErrUnreadable.Error() != "protocol.unreadable-message" {
		t.Fatalf("code %q", ErrUnreadable.Error())
	}
}

func TestASnapshotWithTypeLastIsStillRead(t *testing.T) {
	_, states, err := Read(`{"players":[{"id":"b","p":[1,0,0],"q":[0,0,0,1],"v":[0,0,0]}],"type":"snapshot"}`)
	if err != nil || len(states) != 1 || states[0].Id != "b" {
		t.Fatalf("got %+v, %v", states, err)
	}
}
