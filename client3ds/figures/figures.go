// Package figures measures how a game runs, for the console's Figures tab: frames a second, time
// spent on the processor and the GPU, what was drawn, and what went through the network. Each
// figure is gathered over a second and shown for the next, so the numbers stay readable.
package figures

// Frame is what one frame measured.
type Frame struct {
	Seconds               float64 // since the previous frame
	ProcessorMilliseconds float32 // the game and the planning, on the processor
	NetworkMilliseconds   float32 // of which reading and answering the server
	DrawingMilliseconds   float32 // of which preparing the drawing (baking, planning, the touch screen)
	GPUMilliseconds       float32 // drawing, on the GPU (citro3d's measure)
	Triangles             int
	DrawCalls             int
	Bakes                 int // light worked out again on the processor, since the renderer was made
	HeapBytes             uint64
	Players               int
	SnapshotDelay         float64 // seconds since the server's last snapshot
}

// Second is a second's figures, averaged or summed.
type Second struct {
	FramesPerSecond           int
	ProcessorMilliseconds     float32 // per frame, on average
	NetworkMilliseconds       float32
	DrawingMilliseconds       float32
	GPUMilliseconds           float32
	SlowestFrameMilliseconds  float32
	Triangles                 int // in the last frame
	DrawCalls                 int
	BakesPerSecond            int
	HeapBytes                 uint64
	Players                   int
	MessagesIn                int // a second
	MessagesOut               int
	BytesIn, BytesOut         int // a second
	SnapshotDelayMilliseconds float32
}

// millisecondsPerSecond turns seconds into milliseconds.
const millisecondsPerSecond = 1000

// Counter gathers figures; Shown is the last whole second.
type Counter struct {
	Shown Second

	current               Second
	elapsed               float64
	firstBakes            int
	started               bool
	processorMilliseconds float32
	networkMilliseconds   float32
	drawingMilliseconds   float32
	gpuMilliseconds       float32
}

// AddFrame counts one frame.
func (counter *Counter) AddFrame(frame Frame) {
	if !counter.started {
		counter.firstBakes, counter.started = frame.Bakes, true
	}
	counter.current.FramesPerSecond++
	counter.elapsed += frame.Seconds
	counter.processorMilliseconds += frame.ProcessorMilliseconds
	counter.networkMilliseconds += frame.NetworkMilliseconds
	counter.drawingMilliseconds += frame.DrawingMilliseconds
	counter.gpuMilliseconds += frame.GPUMilliseconds
	counter.current.SlowestFrameMilliseconds = max(counter.current.SlowestFrameMilliseconds, float32(frame.Seconds*millisecondsPerSecond))
	counter.current.Triangles, counter.current.DrawCalls = frame.Triangles, frame.DrawCalls
	counter.current.HeapBytes, counter.current.Players = frame.HeapBytes, frame.Players
	counter.current.SnapshotDelayMilliseconds = float32(frame.SnapshotDelay * millisecondsPerSecond)
	if counter.elapsed < 1 {
		return
	}
	frames := float32(counter.current.FramesPerSecond)
	counter.current.ProcessorMilliseconds, counter.current.GPUMilliseconds = counter.processorMilliseconds/frames, counter.gpuMilliseconds/frames
	counter.current.NetworkMilliseconds, counter.current.DrawingMilliseconds = counter.networkMilliseconds/frames, counter.drawingMilliseconds/frames
	counter.current.BakesPerSecond = frame.Bakes - counter.firstBakes
	counter.Shown = counter.current
	counter.current = Second{}
	counter.elapsed, counter.processorMilliseconds, counter.gpuMilliseconds, counter.firstBakes = 0, 0, 0, frame.Bakes
	counter.networkMilliseconds, counter.drawingMilliseconds = 0, 0
}

// Received counts a message from the server.
func (counter *Counter) Received(bytes int) {
	counter.current.MessagesIn++
	counter.current.BytesIn += bytes
}

// Sent counts a message to the server.
func (counter *Counter) Sent(bytes int) {
	counter.current.MessagesOut++
	counter.current.BytesOut += bytes
}
