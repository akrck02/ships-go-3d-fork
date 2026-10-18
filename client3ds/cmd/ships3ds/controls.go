//go:build nintendo3ds

package main

import (
	"github.com/akrck02/go-renderer/ctr"

	"ships3d/client3ds/flight"
)

// controlsOf reads the stick and the buttons as the flight model's controls.
func controlsOf(reading ctr.Reading) flight.Controls {
	circleX, circleY := reading.Circle()
	controls := flight.Controls{Pitch: -circleY, Yaw: circleX}
	if reading.Held&ctr.ButtonL != 0 {
		controls.Roll--
	}
	if reading.Held&ctr.ButtonR != 0 {
		controls.Roll++
	}
	if reading.Held&ctr.ButtonX != 0 {
		controls.Throttle++
	}
	if reading.Held&ctr.ButtonB != 0 {
		controls.Throttle--
	}
	return controls
}
