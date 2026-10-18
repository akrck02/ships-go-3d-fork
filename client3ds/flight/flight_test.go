package flight

import (
	"math"
	"testing"
)

var settings = Settings{WorldRadius: 1500, MaxSpeed: 80, MinSpeed: -20, Acceleration: 40, PitchRate: 1.4, YawRate: 1, RollRate: 2.2}

const tolerance = 1e-9

func near(first, second Vector) bool { return first.Subtract(second).Length() < tolerance }

func TestRaisingTheNoseClimbs(t *testing.T) {
	state := State{Rotation: NoRotation, Speed: 50}
	for range 10 {
		Step(&state, Controls{Pitch: 1}, settings, 0.05)
	}
	if state.Position[1] <= 0 {
		t.Fatalf("height %v after pulling up, want above 0", state.Position[1])
	}
}

func TestTurningRightGoesTowardMinusX(t *testing.T) {
	state := State{Rotation: NoRotation, Speed: 50}
	for range 10 {
		Step(&state, Controls{Yaw: 1}, settings, 0.05)
	}
	if state.Position[0] >= 0 {
		t.Fatalf("x %v after turning right, want below 0 (+X is the ship's left)", state.Position[0])
	}
}

func TestSpeedStaysWithinTheSettings(t *testing.T) {
	state := State{Rotation: NoRotation}
	Step(&state, Controls{Throttle: 1}, settings, 100)
	if state.Speed != settings.MaxSpeed {
		t.Fatalf("speed %v, want %v", state.Speed, settings.MaxSpeed)
	}
	Step(&state, Controls{Throttle: -1}, settings, 100)
	if state.Speed != settings.MinSpeed {
		t.Fatalf("speed %v, want %v", state.Speed, settings.MinSpeed)
	}
}

func TestTheWorldEdgeIsAWall(t *testing.T) {
	state := State{Position: Vector{0, 0, 1490}, Rotation: NoRotation, Speed: 80}
	Step(&state, Controls{}, settings, 1)
	if length := state.Position.Length(); math.Abs(length-settings.WorldRadius) > tolerance {
		t.Fatalf("distance %v, want the world's radius", length)
	}
}

func TestAQuarterTurnAboutXPointsTheNoseDown(t *testing.T) {
	if got := FromEulerXYZ(math.Pi/2, 0, 0).Apply(Forward); !near(got, Vector{0, -1, 0}) {
		t.Fatalf("got %v", got)
	}
}

func TestEulerAnglesTurnXThenYThenZInTheMovingFrame(t *testing.T) {
	combined := FromEulerXYZ(0.3, 0.5, 0.7)
	stepByStep := FromEulerXYZ(0.3, 0, 0).Then(FromEulerXYZ(0, 0.5, 0)).Then(FromEulerXYZ(0, 0, 0.7))
	probe := Vector{1, 2, 3}
	if !near(combined.Apply(probe), stepByStep.Apply(probe)) {
		t.Fatalf("%v and %v differ", combined.Apply(probe), stepByStep.Apply(probe))
	}
}

func TestTowardReachesItsTarget(t *testing.T) {
	target := FromEulerXYZ(0, 2, 0)
	if got := NoRotation.Toward(target, 1).Apply(Forward); !near(got, target.Apply(Forward)) {
		t.Fatalf("got %v", got)
	}
}

func TestABulletPassingThroughAShipHitsIt(t *testing.T) {
	centre := Vector{0, 0, 50}
	if !SegmentHitsSphere(Vector{0, 1, 0}, Vector{0, 1, 100}, centre, 4) {
		t.Fatal("a path through the ship missed")
	}
	if SegmentHitsSphere(Vector{10, 0, 0}, Vector{10, 0, 100}, centre, 4) {
		t.Fatal("a path beside the ship hit")
	}
	if SegmentHitsSphere(Vector{0, 0, 0}, Vector{0, 0, 40}, centre, 4) {
		t.Fatal("a path that stops short hit")
	}
}
