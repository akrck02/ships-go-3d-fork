//go:build nintendo3ds

package main

import (
	"github.com/akrck02/go-3ds/app"
	"github.com/akrck02/go-renderer/ctr"
	ui "github.com/akrck02/go-ui"

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

// tiltOf adds a frame's turning of the console up, on the gyroscope's axes: its pitch tips the
// top screen, its roll turns the console like a wheel.
// TODO(akrck02): check on the console that both signs feel right; the emulator has no gyroscope.
func tiltOf(tilted *tilt.Tilt, frame app.Frame) {
	tilted.Turn(frame.Gyroscope.Pitch, frame.Gyroscope.Roll, frame.Seconds)
}

// controlLines are what each control does, for the Controls tab.
var controlLines = [][2]string{
	{"Circle pad", "Down raises the nose; turn"},
	{"R or A", "Fire; A flies again after dying"},
	{"C-stick", "Roll, faster and slower"},
	{"Left / Right", "Roll"},
	{"ZR / ZL", "Faster / slower (also X / B)"},
	{"Tilting", "Nose and turn, if on in Settings"},
	{"L", "Centre the tilt"},
	{"Left / Right", "Ship to fly, on the ship screen"},
	{"Up / Down", "Radar zoom"},
	{"3D slider", "Depth"},
	{"START", "Options"},
}

// buildControls lists the controls, two columns, scrolled with a finger when they don't fit.
func buildControls(context *ui.Context) {
	context.ScrollArea("controls", ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow()}, func() {
		buildControlLines(context)
	})
}

func buildControlLines(context *ui.Context) {
	for _, line := range controlLines {
		context.Row(func() {
			context.Box(ui.Layout{Width: ui.Fixed(controlNameWidth)}, func() {
				context.Text(line[0], ui.TextStyle{Weight: boldWeight})
			})
			context.Text(line[1], ui.TextStyle{})
		})
	}
}

// controlNameWidth is the first column, in pixels.
const controlNameWidth = 92
