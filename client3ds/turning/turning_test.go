package turning

import (
	"math"
	"testing"

	"ships3d/client3ds/flight"
)

func TestNoMoreThanTheBudgetIsChosen(t *testing.T) {
	var chooser Chooser
	candidates := make([]Candidate, 30)
	for index := range candidates {
		candidates[index] = Candidate{Turned: 0.1, Distance: 100}
	}
	if got := chooser.Choose(candidates, 3); len(got) != 3 {
		t.Fatalf("chose %d, want 3", len(got))
	}
}

func TestTheNearestAndMostTurnedGoFirst(t *testing.T) {
	var chooser Chooser
	candidates := []Candidate{
		{Turned: 0.1, Distance: 500}, // far, a little turned
		{Turned: 0.5, Distance: 50},  // near and turned a lot
		{Turned: 0.1, Distance: 50},  // near, a little
	}
	got := chooser.Choose(candidates, 2)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("chose %v, want [1 2]", got)
	}
}

func TestShipsThatDidntTurnAreLeftAlone(t *testing.T) {
	var chooser Chooser
	if got := chooser.Choose([]Candidate{{Distance: 5}, {Distance: 5}}, 5); len(got) != 0 {
		t.Fatalf("chose %v, want none", got)
	}
}

func TestAChangeOfDetailIsChosenEvenWithoutTurning(t *testing.T) {
	var chooser Chooser
	if got := chooser.Choose([]Candidate{{Distance: 200, Detail: true}}, 1); len(got) != 1 {
		t.Fatalf("chose %v", got)
	}
}

func TestTheAngleBetweenRotations(t *testing.T) {
	quarter := flight.FromEulerXYZ(0, math.Pi/2, 0)
	if got := Angle(flight.NoRotation, quarter); math.Abs(got-math.Pi/2) > 1e-9 {
		t.Fatalf("got %v, want a quarter turn", got)
	}
	opposite := flight.Rotation{-quarter[0], -quarter[1], -quarter[2], -quarter[3]} // the same rotation
	if got := Angle(quarter, opposite); got > 1e-6 {
		t.Fatalf("got %v, want 0", got)
	}
}

func TestChoosingAllocatesNothingOnceWarm(t *testing.T) {
	var chooser Chooser
	candidates := []Candidate{{Turned: 0.1, Distance: 50}, {Turned: 0.2, Distance: 60}}
	chooser.Choose(candidates, 1)
	if allocations := testing.AllocsPerRun(100, func() { chooser.Choose(candidates, 1) }); allocations > 0 {
		t.Fatalf("%v allocations a frame", allocations)
	}
}
