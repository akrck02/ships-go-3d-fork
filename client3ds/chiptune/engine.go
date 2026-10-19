package chiptune

import "math/rand/v2"

// The engine's hum: a low triangle and a square an octave and a fifth over it, with a rush of
// noise, all rising with the throttle. Frequencies are in hertz.
const (
	engineIdleFrequency = 42
	engineFullFrequency = 120
	engineIdleVolume    = 0.16
	engineFullVolume    = 0.42
	engineGlide         = 0.0004 // how much of the way to the throttle's pitch each sample goes
	engineNoiseShare    = 0.35
	engineWobbleHertz   = 7 // the hum's slight flutter, as an old engine's
	engineWobbleDepth   = 0.04
)

// Engine is the ship's engine, worked out as it plays: a go-sound stream that never ends. Set
// Throttle, from 0 (idle) to 1 (full), as the ship speeds up and slows down; the pitch glides
// there instead of jumping.
type Engine struct {
	Throttle float64
	Volume   float64
	pitch    float64 // the throttle the sound has glided to
	phases   [3]float64
	rush     float64 // the noise, low-passed
	random   *rand.Rand
}

// NewEngine is an engine at idle, at a volume from 0 to 1.
func NewEngine(volume float64) *Engine {
	return &Engine{Volume: volume, random: rand.New(rand.NewPCG(2, 2))}
}

// Format is the engine's: SampleRate, one channel.
func (engine *Engine) Format() (sampleRate, channels int) { return SampleRate, 1 }

// Rewind does nothing: the engine has no start to go back to.
func (engine *Engine) Rewind() error { return nil }

// Read fills samples with the engine's hum; it never ends.
func (engine *Engine) Read(samples []float32) (int, error) {
	throttle := min(max(engine.Throttle, 0), 1)
	shape := square(0.3)
	for index := range samples {
		engine.pitch += (throttle - engine.pitch) * engineGlide
		wobble := 1 + engineWobbleDepth*sine(engine.phases[2])
		frequency := (engineIdleFrequency + (engineFullFrequency-engineIdleFrequency)*engine.pitch) * wobble
		engine.turn(0, frequency/SampleRate)
		engine.turn(1, 3*frequency/SampleRate)
		engine.turn(2, engineWobbleHertz/SampleRate)
		engine.rush += (0.05 + 0.25*engine.pitch) * ((engine.random.Float64()*2 - 1) - engine.rush)
		hum := 0.6*triangle(engine.phases[0]) + 0.25*shape(engine.phases[1]) + engineNoiseShare*engine.rush
		volume := engineIdleVolume + (engineFullVolume-engineIdleVolume)*engine.pitch
		samples[index] = float32(engine.Volume * volume * hum)
	}
	return len(samples), nil
}

// turn moves a phase on by step, round from 1 to 0: cheaper than math.Mod once a sample.
func (engine *Engine) turn(index int, step float64) {
	engine.phases[index] += step
	if engine.phases[index] >= 1 {
		engine.phases[index]--
	}
}
