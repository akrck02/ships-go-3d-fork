package figures

import "testing"

func TestASecondOfFramesIsShownWhole(t *testing.T) {
	var counter Counter
	for frame := range 60 {
		counter.Received(100)
		if frame%3 == 0 {
			counter.Sent(50)
		}
		counter.AddFrame(Frame{Seconds: 1.0 / 60, ProcessorMilliseconds: 4, GPUMilliseconds: 8, Triangles: 1000, Bakes: frame * 2})
	}
	shown := counter.Shown
	if shown.FramesPerSecond != 60 || shown.ProcessorMilliseconds != 4 || shown.GPUMilliseconds != 8 {
		t.Fatalf("got %+v", shown)
	}
	if shown.MessagesIn != 60 || shown.BytesIn != 6000 || shown.MessagesOut != 20 || shown.BytesOut != 1000 {
		t.Fatalf("network %+v", shown)
	}
	if shown.BakesPerSecond != 118 {
		t.Fatalf("%d bakes, want 118", shown.BakesPerSecond)
	}
}

func TestNothingIsShownBeforeASecond(t *testing.T) {
	var counter Counter
	counter.AddFrame(Frame{Seconds: 0.5})
	if counter.Shown.FramesPerSecond != 0 {
		t.Fatalf("shown %+v after half a second", counter.Shown)
	}
}

func TestTheSlowestFrameIsKept(t *testing.T) {
	var counter Counter
	counter.AddFrame(Frame{Seconds: 0.1})
	counter.AddFrame(Frame{Seconds: 0.9})
	if counter.Shown.SlowestFrameMilliseconds != 900 {
		t.Fatalf("slowest %v ms, want 900", counter.Shown.SlowestFrameMilliseconds)
	}
}
