package flight

// Forward is where a ship's nose points before it turns: +Z, with +Y up, as in the editor.
var Forward = Vector{0, 0, 1}

// Controls are the stick and the throttle, each in [-1, 1]: pitch +1 raises the nose, yaw +1
// turns right, roll +1 rolls right, throttle +1 speeds up.
type Controls struct {
	Pitch, Yaw, Roll, Throttle float64
}

// Settings are the rules of flight, which the server sends in its welcome.
type Settings struct {
	WorldRadius  float64 `json:"worldRadius"`
	MaxSpeed     float64 `json:"maxSpeed"`
	MinSpeed     float64 `json:"minSpeed"`
	Acceleration float64 `json:"acceleration"`
	PitchRate    float64 `json:"pitchRate"`
	YawRate      float64 `json:"yawRate"`
	RollRate     float64 `json:"rollRate"`
}

// State is where a ship is, where it faces and how fast it goes along its nose (negative is
// reverse).
type State struct {
	Position Vector
	Rotation Rotation
	Speed    float64
}

// Step moves a ship on by seconds.
//
// With +Z forward and +Y up in a right-handed frame, +X is the ship's left: raising the nose is a
// negative turn about X, and turning right a negative turn about Y.
func Step(state *State, controls Controls, settings Settings, seconds float64) {
	turn := FromEulerXYZ(
		-clamp(controls.Pitch, -1, 1)*settings.PitchRate*seconds,
		-clamp(controls.Yaw, -1, 1)*settings.YawRate*seconds,
		clamp(controls.Roll, -1, 1)*settings.RollRate*seconds,
	)
	state.Rotation = state.Rotation.Then(turn).Normalized()
	state.Speed = clamp(state.Speed+clamp(controls.Throttle, -1, 1)*settings.Acceleration*seconds, settings.MinSpeed, settings.MaxSpeed)
	state.Position = state.Position.Add(state.Rotation.Apply(Forward).Times(state.Speed * seconds))
	// The edge of the world is a wall: the ship slides along it.
	if distance := state.Position.Length(); distance > settings.WorldRadius {
		state.Position = state.Position.Times(settings.WorldRadius / distance)
	}
}

// Velocity is the ship's velocity.
func (state State) Velocity() Vector { return state.Rotation.Apply(Forward).Times(state.Speed) }

// SegmentHitsSphere tells whether something moving from from to to passed within radius of
// centre. A bullet covers several ship widths a frame, so the whole path is tested, not where it
// ends.
func SegmentHitsSphere(from, to, centre Vector, radius float64) bool {
	path := to.Subtract(from)
	lengthSquared := path.Dot(path)
	// A path shorter than this is a point: dividing by its length would only add noise.
	const pointLengthSquared = 1e-12
	share := 0.0
	if lengthSquared > pointLengthSquared {
		share = clamp(centre.Subtract(from).Dot(path)/lengthSquared, 0, 1)
	}
	closest := from.Add(path.Times(share)).Subtract(centre)
	return closest.Dot(closest) <= radius*radius
}

func clamp(value, minimum, maximum float64) float64 {
	return min(maximum, max(minimum, value))
}
