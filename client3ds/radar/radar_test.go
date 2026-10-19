package radar

import (
	"math"
	"testing"

	"ships3d/client3ds/flight"
)

const tolerance = 1e-9

func TestAShipAheadIsUpOnTheMap(t *testing.T) {
	blip := Place(flight.Vector{0, 0, 300}, flight.Vector{}, flight.NoRotation, 600)
	if !blip.InRange || math.Abs(blip.Ahead-0.5) > tolerance || math.Abs(blip.Right) > tolerance {
		t.Fatalf("got %+v", blip)
	}
}

func TestTheMapTurnsWithTheShip(t *testing.T) {
	// Turned right by a quarter turn, a ship that was to our right is now ahead.
	turnedRight := flight.FromEulerXYZ(0, -math.Pi/2, 0)
	blip := Place(flight.Vector{-100, 0, 0}, flight.Vector{}, turnedRight, 100)
	if math.Abs(blip.Ahead-1) > tolerance || math.Abs(blip.Right) > tolerance {
		t.Fatalf("got %+v", blip)
	}
}

func TestAFarShipIsPinnedToTheEdge(t *testing.T) {
	blip := Place(flight.Vector{0, 3000, 4000}, flight.Vector{}, flight.NoRotation, 600)
	if blip.InRange || math.Abs(blip.Ahead-0.8) > tolerance || math.Abs(blip.Above-0.6) > tolerance {
		t.Fatalf("got %+v", blip)
	}
}

func TestABlipOnTheRightIsOnMinusX(t *testing.T) {
	x, y, z := Blip{Right: 0.5, Ahead: 0.2, Above: 1}.Scene()
	if x != -0.5 || z != 0.2 || y != heightShare {
		t.Fatalf("got %v %v %v", x, y, z)
	}
}
