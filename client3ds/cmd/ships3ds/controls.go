//go:build nintendo3ds

package main

import (
	"github.com/akrck02/go-3ds/app"
	"github.com/akrck02/go-renderer/ctr"
	ui "github.com/akrck02/go-ui"

	"ships3d/client3ds/flight"
)

// controlsOf reads the sticks and the buttons as the flight model's controls. The C-stick (New
// 3DS) rolls and changes speed, as L, R, X and B do; both can be used at once.
func controlsOf(frame app.Frame) flight.Controls {
	circleX, circleY := frame.Reading.Circle()
	controls := flight.Controls{Pitch: -circleY, Yaw: circleX, Roll: frame.CStickX, Throttle: frame.CStickY}
	held := frame.Reading.Held
	if held&ctr.ButtonL != 0 {
		controls.Roll--
	}
	if held&ctr.ButtonR != 0 {
		controls.Roll++
	}
	if held&ctr.ButtonX != 0 {
		controls.Throttle++
	}
	if held&ctr.ButtonB != 0 {
		controls.Throttle--
	}
	return controls
}

// controlLines are what each control does, for the Controls tab.
var controlLines = [][2]string{
	{"Circle pad", "Nose up and down, turn"},
	{"C-stick", "Roll, faster and slower"},
	{"L / R", "Roll left / right"},
	{"X / B", "Faster / slower"},
	{"A", "Fire; fly again after dying"},
	{"Up / Down", "Radar range"},
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
