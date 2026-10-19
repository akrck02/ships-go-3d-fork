//go:build nintendo3ds

package main

import (
	"math"
	"strconv"

	"github.com/akrck02/go-renderer/draw2d"
)

// The frame budget the figures are judged against: the screen's 60 frames a second.
const (
	targetFramesPerSecond   = 60
	frameBudgetMilliseconds = 1000.0 / targetFramesPerSecond
	bytesPerKilobyte        = 1024
	bytesPerMegabyte        = 1024 * 1024
)

// figuresPage shows how the game runs: the frame, the drawing, the network and the tilt. A figure
// within its budget shows gold, one over it red.
func (session *session) figuresPage() menuPage {
	shown := session.figures.Shown
	judged := func(good bool) draw2d.Color {
		if good {
			return pixel.gold
		}
		return pixel.danger
	}
	snapshotBudget := 2 * 1000 / float32(max(session.game.Settings.TickRate, 1))
	gyroscope := session.gyroscope
	return menuPage{title: figuresPage, valueLeft: 150, rows: []menuRow{
		{label: "Frames", value: strconv.Itoa(shown.FramesPerSecond) + " a second", colour: judged(shown.FramesPerSecond >= targetFramesPerSecond-2)},
		{label: "Processor", value: tenths(shown.ProcessorMilliseconds) + " ms", colour: judged(shown.ProcessorMilliseconds < frameBudgetMilliseconds)},
		{label: "Graphics", value: tenths(shown.GPUMilliseconds) + " ms", colour: judged(shown.GPUMilliseconds < frameBudgetMilliseconds)},
		{label: "Slowest frame", value: tenths(shown.SlowestFrameMilliseconds) + " ms", colour: judged(shown.SlowestFrameMilliseconds < 2*frameBudgetMilliseconds)},
		{label: "Network", value: tenths(shown.NetworkMilliseconds) + " ms"},
		{label: "Drawing", value: tenths(shown.DrawingMilliseconds) + " ms"},
		{label: "Heap", value: tenths(float32(shown.HeapBytes)/bytesPerMegabyte) + " MB"},
		{label: "Triangles", value: strconv.Itoa(shown.Triangles)},
		{label: "Draws", value: strconv.Itoa(shown.DrawCalls)},
		{label: "Bakes", value: strconv.Itoa(shown.BakesPerSecond) + " a second"},
		{label: "Received", value: strconv.Itoa(shown.MessagesIn) + " msg, " + tenths(float32(shown.BytesIn)/bytesPerKilobyte) + " KB/s"},
		{label: "Sent", value: strconv.Itoa(shown.MessagesOut) + " msg, " + tenths(float32(shown.BytesOut)/bytesPerKilobyte) + " KB/s"},
		{label: "Since snapshot", value: strconv.Itoa(int(shown.SnapshotDelayMilliseconds)) + " ms", colour: judged(shown.SnapshotDelayMilliseconds < snapshotBudget)},
		{label: "Players", value: strconv.Itoa(shown.Players)},
		{label: "Gyroscope", value: signed(gyroscope.Roll) + " " + signed(gyroscope.Pitch) + " " + signed(gyroscope.Yaw) + " deg/s"},
		{label: "Tilt / steer", value: signed(session.tilt.Pitch) + " / " + signed(session.tilt.Steer)},
	}}
}

// tenths writes a number with one decimal.
func tenths(value float32) string {
	whole := int(value*10 + 0.5)
	return strconv.Itoa(whole/10) + "." + strconv.Itoa(whole%10)
}

// signed writes a whole number with its sign.
func signed(value float64) string {
	rounded := int(math.Round(value))
	if rounded > 0 {
		return "+" + strconv.Itoa(rounded)
	}
	return strconv.Itoa(rounded)
}
