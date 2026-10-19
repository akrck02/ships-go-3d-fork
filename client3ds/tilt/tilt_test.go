package tilt

import (
	"math"
	"testing"
)

func TestTurningIsAddedUp(t *testing.T) {
	var tilt Tilt
	for range 60 {
		tilt.Turn(10, -5, 1.0/60)
	}
	if math.Abs(tilt.Pitch-10) > 1e-9 || math.Abs(tilt.Steer+5) > 1e-9 {
		t.Fatalf("got %+v after a second", tilt)
	}
	tilt.Centre()
	if tilt != (Tilt{}) {
		t.Fatalf("got %+v after centring", tilt)
	}
}

func TestTheAxisHasADeadZoneAndAFullTilt(t *testing.T) {
	cases := map[float64]float64{0: 0, 2.9: 0, -3: 0, 25: 1, 40: 1, -25: -1}
	for degrees, want := range cases {
		if got := Axis(degrees); got != want {
			t.Errorf("Axis(%v) = %v, want %v", degrees, got, want)
		}
	}
	if half := Axis(14); half <= 0 || half >= 0.5 {
		t.Errorf("half way gives %v, want a gentle start", half)
	}
}
