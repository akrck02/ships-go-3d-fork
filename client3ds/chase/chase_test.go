package chase

import (
	"testing"

	"ships3d/client3ds/flight"
)

func TestTheCameraStartsBehindTheShip(t *testing.T) {
	var camera Camera
	view := camera.Follow(flight.State{Rotation: flight.NoRotation}, 4, 1.0/60)
	if view.Eye[2] >= 0 || view.Eye[1] <= 0 || view.Target[2] <= 0 {
		t.Fatalf("view %+v, want above and behind, looking ahead", view)
	}
}

func TestTheCameraLagsThenCatchesUp(t *testing.T) {
	var camera Camera
	camera.Follow(flight.State{Rotation: flight.NoRotation}, 4, 1.0/60)
	moved := flight.State{Position: flight.Vector{0, 0, 100}, Rotation: flight.NoRotation}
	first := camera.Follow(moved, 4, 1.0/60)
	if first.Eye[2] > 50 {
		t.Fatalf("eye at %v after one frame, want still near the start", first.Eye[2])
	}
	for range 120 {
		camera.Follow(moved, 4, 1.0/60)
	}
	if camera.Position.Subtract(moved.Position).Length() > 0.01 {
		t.Fatalf("still %v away after two seconds", camera.Position.Subtract(moved.Position).Length())
	}
}

func TestResetJumpsToTheShip(t *testing.T) {
	var camera Camera
	camera.Follow(flight.State{Rotation: flight.NoRotation}, 4, 1.0/60)
	camera.Reset()
	camera.Follow(flight.State{Position: flight.Vector{500, 0, 0}, Rotation: flight.NoRotation}, 4, 1.0/60)
	if camera.Position[0] != 500 {
		t.Fatalf("camera at %v, want at the ship", camera.Position)
	}
}

func TestTheViewSeesAheadAndNotBehind(t *testing.T) {
	view := View{Eye: flight.Vector{}, Target: flight.Vector{0, 0, 1}, Up: flight.Vector{0, 1, 0}}
	cases := map[string]struct {
		centre flight.Vector
		want   bool
	}{
		"straight ahead":           {flight.Vector{0, 0, 100}, true},
		"behind":                   {flight.Vector{0, 0, -100}, false},
		"far out to the side":      {flight.Vector{100, 0, 10}, false},
		"at the edge, but big":     {flight.Vector{60, 0, 100}, true},
		"beyond the farthest":      {flight.Vector{0, 0, 900}, false},
		"around the camera itself": {flight.Vector{0, 0, -1}, true},
	}
	for name, test := range cases {
		if got := view.Sees(test.centre, 4, 600); got != test.want {
			t.Errorf("%s: got %v, want %v", name, got, test.want)
		}
	}
}
