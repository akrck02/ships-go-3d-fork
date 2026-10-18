//go:build nintendo3ds

package main

import (
	"strconv"

	ui "github.com/akrck02/go-ui"
)

// The frame budget the figures are judged against: the screen's 60 frames a second.
const (
	targetFramesPerSecond   = 60
	frameBudgetMilliseconds = 1000.0 / targetFramesPerSecond
	bytesPerKilobyte        = 1024
	bytesPerMegabyte        = 1024 * 1024
)

// buildFigures shows how the game runs: the frame, the drawing and the network, in chips.
func (session *session) buildFigures(context *ui.Context) {
	theme := context.Theme
	plain := ui.ChipLook{Fill: theme.Sunken, Ink: theme.InkSoft}
	judged := func(good bool) ui.ChipLook {
		if good {
			return ui.ChipLook{Fill: theme.PrimaryTint, Ink: theme.Primary}
		}
		return ui.ChipLook{Fill: theme.Sunken, Ink: theme.Warning}
	}
	context.ScrollArea("figures", ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow(), Gap: figuresGap}, func() {
		session.buildFigureChips(context, judged, plain)
	})
}

// figuresGap is the space between the Figures tab's rows, in pixels.
const figuresGap = 4

func (session *session) buildFigureChips(context *ui.Context, judged func(bool) ui.ChipLook, plain ui.ChipLook) {
	shown := session.figures.Shown
	context.Caption("Frame")
	context.Row(func() {
		context.Chip("gauge", strconv.Itoa(shown.FramesPerSecond)+" fps", judged(shown.FramesPerSecond >= targetFramesPerSecond-2))
		context.Chip("cpu", tenths(shown.ProcessorMilliseconds)+" ms CPU", judged(shown.ProcessorMilliseconds < frameBudgetMilliseconds))
		context.Chip("monitor", tenths(shown.GPUMilliseconds)+" ms GPU", judged(shown.GPUMilliseconds < frameBudgetMilliseconds))
	})
	context.Row(func() {
		context.Chip("timer", tenths(shown.SlowestFrameMilliseconds)+" ms slowest", judged(shown.SlowestFrameMilliseconds < 2*frameBudgetMilliseconds))
		context.Chip("memory-stick", tenths(float32(shown.HeapBytes)/bytesPerMegabyte)+" MB heap", plain)
	})
	context.Caption("Drawing")
	context.Row(func() {
		context.Chip("triangle", strconv.Itoa(shown.Triangles)+" triangles", plain)
		context.Chip("layers", strconv.Itoa(shown.DrawCalls)+" draws", plain)
		context.Chip("sun", strconv.Itoa(shown.BakesPerSecond)+" bakes/s", plain)
	})
	context.Caption("Network")
	context.Row(func() {
		context.Chip("arrow-down", strconv.Itoa(shown.MessagesIn)+" msg · "+tenths(float32(shown.BytesIn)/bytesPerKilobyte)+" KB/s", plain)
		context.Chip("arrow-up", strconv.Itoa(shown.MessagesOut)+" msg · "+tenths(float32(shown.BytesOut)/bytesPerKilobyte)+" KB/s", plain)
	})
	context.Row(func() {
		context.Chip("radio", strconv.Itoa(int(shown.SnapshotDelayMilliseconds))+" ms since snapshot", judged(shown.SnapshotDelayMilliseconds < 2*1000/float32(max(session.game.Settings.TickRate, 1))))
		context.Chip("users", strconv.Itoa(shown.Players)+" players", plain)
	})
}

// tenths writes a number with one decimal.
func tenths(value float32) string {
	whole := int(value*10 + 0.5)
	return strconv.Itoa(whole/10) + "." + strconv.Itoa(whole%10)
}
