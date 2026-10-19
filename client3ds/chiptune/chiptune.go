// Package chiptune makes the game's sounds the way handheld consoles did: square and triangle
// waves and noise, shaped by envelopes, worked out in code rather than read from files. Effects
// are short buffers made once; the music is worked out as it plays.
package chiptune

import (
	"io"
	"math"
	"math/rand/v2"
)

// SampleRate is the sounds' frames a second: enough for chiptune, and light on the processor.
const SampleRate = 32000

// Sound is one sound ready to play again and again: mono samples from -1 to 1. It is a go-sound
// stream (Format and Read) that can rewind.
type Sound struct {
	samples  []float32
	position int
}

// Format is the sound's: SampleRate, one channel. (go-sound's sound.Format has these fields.)
func (sound *Sound) Format() (sampleRate, channels int) { return SampleRate, 1 }

// Read fills samples from where the sound is; io.EOF at its end.
func (sound *Sound) Read(samples []float32) (int, error) {
	if sound.position >= len(sound.samples) {
		return 0, io.EOF
	}
	count := copy(samples, sound.samples[sound.position:])
	sound.position += count
	return count, nil
}

// Rewind starts the sound again.
func (sound *Sound) Rewind() error {
	sound.position = 0
	return nil
}

// Seconds is how long the sound lasts.
func (sound *Sound) Seconds() float64 { return float64(len(sound.samples)) / SampleRate }

// Copy is another sound with the same samples, to play over the first.
func (sound *Sound) Copy() *Sound { return &Sound{samples: sound.samples} }

// wave is a shape repeated at a frequency, given the phase from 0 to 1.
type wave func(phase float64) float64

func square(duty float64) wave {
	return func(phase float64) float64 {
		if phase < duty {
			return 1
		}
		return -1
	}
}

func triangle(phase float64) float64 { return 4*math.Abs(phase-0.5) - 1 }

func sine(phase float64) float64 { return math.Sin(2 * math.Pi * phase) }

// sweep is a note whose frequency slides from one to another while its loudness falls.
func sweep(shape wave, from, to, seconds, volume float64) []float32 {
	count := int(seconds * SampleRate)
	samples := make([]float32, count)
	phase := 0.0
	for index := range samples {
		share := float64(index) / float64(count)
		frequency := from * math.Pow(to/from, share) // even in pitch, as the ear hears
		phase = math.Mod(phase+frequency/SampleRate, 1)
		samples[index] = float32(volume * shape(phase) * decay(share))
	}
	return samples
}

// decay is an envelope from 1 to 0, quick at first, with a short rise so it doesn't click.
func decay(share float64) float64 {
	const rise = 0.02
	if share < rise {
		return share / rise
	}
	left := 1 - share
	return left * left
}

// noise is a burst of noise whose brightness falls with its loudness: low-passed more and more,
// as an explosion's crack turns into a rumble.
func noise(seconds, volume, brightness float64, seed uint64) []float32 {
	random := rand.New(rand.NewPCG(seed, seed))
	count := int(seconds * SampleRate)
	samples := make([]float32, count)
	filtered := 0.0
	for index := range samples {
		share := float64(index) / float64(count)
		smoothing := brightness * (1 - 0.9*share) // how much of each new sample comes through
		filtered += smoothing * ((random.Float64()*2 - 1) - filtered)
		samples[index] = float32(volume * filtered * decay(share))
	}
	return samples
}

// mix adds sounds together, sample by sample, keeping each within -1 and 1.
func mix(parts ...[]float32) []float32 {
	longest := 0
	for _, part := range parts {
		longest = max(longest, len(part))
	}
	mixed := make([]float32, longest)
	for _, part := range parts {
		for index, sample := range part {
			mixed[index] += sample
		}
	}
	for index, sample := range mixed {
		mixed[index] = float32(math.Max(-1, math.Min(1, float64(sample))))
	}
	return mixed
}

// The game's effects.

// Laser is a shot: a square wave diving from high to low.
func Laser() *Sound { return &Sound{samples: sweep(square(0.25), 1600, 260, 0.14, 0.32)} }

// Hit is a bullet striking a ship: a crack of noise over a short low thump.
func Hit() *Sound {
	return &Sound{samples: mix(noise(0.09, 0.5, 0.9, 3), sweep(triangle, 220, 70, 0.12, 0.6))}
}

// Explosion is a ship blowing up: a long rumble of noise and a falling growl.
func Explosion() *Sound {
	return &Sound{samples: mix(noise(1.1, 0.75, 0.6, 7), sweep(square(0.5), 160, 35, 0.8, 0.25))}
}

// Shield is the force field answering a crash: a deep bump and a bright shimmer falling away.
func Shield() *Sound {
	return &Sound{samples: mix(sweep(sine, 150, 60, 0.35, 0.7), sweep(triangle, 1400, 500, 0.5, 0.18))}
}

// Move is the cursor moving in a menu.
func Move() *Sound { return &Sound{samples: sweep(square(0.5), 880, 880, 0.04, 0.22)} }

// Choose is a menu's choice taken: two rising notes.
func Choose() *Sound {
	return &Sound{samples: append(sweep(square(0.5), 660, 660, 0.05, 0.22), sweep(square(0.5), 990, 990, 0.08, 0.22)...)}
}

// Back is leaving a menu: two falling notes.
func Back() *Sound {
	return &Sound{samples: append(sweep(square(0.5), 990, 990, 0.05, 0.2), sweep(square(0.5), 600, 600, 0.08, 0.2)...)}
}
