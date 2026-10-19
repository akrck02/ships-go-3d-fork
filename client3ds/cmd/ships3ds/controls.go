//go:build nintendo3ds

package main

import (
	"github.com/akrck02/go-3ds/app"
	"github.com/akrck02/go-renderer/ctr"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/profile"
	"ships3d/client3ds/tilt"
)

// fireButtons shoot: R, as a trigger, and A.
const fireButtons = ctr.ButtonR | ctr.ButtonA

// controlsOf reads the sticks, the buttons and, when chosen, the console's tilt as the flight
// model's controls. Roll is on the C-stick (New 3DS) and the cross's left and right; speed on the
// C-stick's up and down, ZR and ZL (New 3DS), X and B. They add up, so any can be used at once.
func controlsOf(frame app.Frame, choices profile.Settings, tilted tilt.Tilt) flight.Controls {
	circleX, circleY := frame.Reading.Circle()
	precision := float64(choices.Precision)
	pitch := -flight.Curve(circleY, precision)
	if choices.InvertPitch {
		pitch = -pitch
	}
	controls := flight.Controls{Pitch: pitch, Yaw: flight.Curve(circleX, precision), Roll: frame.CStickX, Throttle: frame.CStickY}
	if choices.Gyroscope {
		controls.Pitch += tilt.Axis(tilted.Pitch)
		controls.Yaw += tilt.Axis(tilted.Steer)
	}
	held := frame.Reading.Held
	if held&ctr.ButtonLeft != 0 {
		controls.Roll--
	}
	if held&ctr.ButtonRight != 0 {
		controls.Roll++
	}
	if held&(ctr.ButtonZR|ctr.ButtonX) != 0 {
		controls.Throttle++
	}
	if held&(ctr.ButtonZL|ctr.ButtonB) != 0 {
		controls.Throttle--
	}
	return controls
}

// tiltOf adds a frame's turning of the console up. libctru's names for the gyroscope's axes are
// the other way round from how the console is held to play: what it calls roll tips the top
// screen (toward the player is negative), what it calls pitch turns the console like a wheel.
// Found on a New 3DS XL: before this, tilting left lowered the nose and tipping up turned left.
func tiltOf(tilted *tilt.Tilt, frame app.Frame) {
	tilted.Turn(-frame.Gyroscope.Roll, frame.Gyroscope.Pitch, frame.Seconds)
}

// controlLines are what each control does, for the Controls tab.
var controlLines = [][2]string{
	{"Circle pad", "Down raises the nose; turn"},
	{"R or A", "Fire; A flies again after dying"},
	{"C-stick", "Roll, faster and slower"},
	{"Left / Right", "Roll"},
	{"ZR / ZL", "Faster / slower (also X / B)"},
	{"Tilting", "Nose and turn, if on in Setup"},
	{"L", "Centre the tilt"},
	{"Left / Right", "Ship to fly, on the ship screen"},
	{"Up / Down", "Radar zoom"},
	{"3D slider", "Depth"},
	{"START", "Options"},
}

// controlNameWidth is the first column, in pixels.
const controlNameWidth = 92
