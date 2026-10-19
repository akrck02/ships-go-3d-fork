package chiptune

import (
	"math"
	"math/rand/v2"
)

// Music is a loop in the style of a handheld shooter: a triangle bass, a square arpeggio over a
// minor progression, and a noise hi-hat, worked out sample by sample as it plays, so it costs no
// memory. It never ends; Rewind starts it again.
type Music struct {
	frame  int
	phases [2]float64 // the bass's and the arpeggio's
	hat    float64    // the hi-hat's noise, low-passed
	random *rand.Rand
	Volume float64
}

// The music's tempo and progression: four chords of two bars each, in A minor.
const (
	beatsPerMinute = 128
	stepsPerBeat   = 4 // sixteenth notes
	stepsPerChord  = 32
)

// chords are the progression's roots, as semitones from A (Am, F, C, G), and each chord's notes.
var chords = [][3]int{{0, 3, 7}, {-4, 0, 3}, {3, 7, 10}, {-2, 2, 5}}

// NewMusic is the loop at a volume from 0 to 1.
func NewMusic(volume float64) *Music {
	return &Music{random: rand.New(rand.NewPCG(1, 1)), Volume: volume}
}

// Format is the music's: SampleRate, one channel.
func (music *Music) Format() (sampleRate, channels int) { return SampleRate, 1 }

// Rewind starts the music again.
func (music *Music) Rewind() error {
	music.frame, music.phases, music.hat = 0, [2]float64{}, 0
	return nil
}

// Read fills samples; the music never ends.
func (music *Music) Read(samples []float32) (int, error) {
	stepFrames := SampleRate * 60 / (beatsPerMinute * stepsPerBeat)
	for index := range samples {
		step := music.frame / stepFrames
		inStep := float64(music.frame%stepFrames) / float64(stepFrames)
		chord := chords[(step/stepsPerChord)%len(chords)]
		// The bass plays the root on each beat, an octave down on the off-beats.
		bassNote := chord[0] - 24
		if (step/2)%2 == 1 {
			bassNote -= 12
		}
		bass := music.voice(0, frequencyOf(bassNote), triangle) * 0.45 * (1 - 0.5*inStep)
		// The arpeggio climbs the chord's notes, one each sixteenth, an octave up every other bar.
		arpeggioNote := chord[step%3] + 12*((step/16)%2)
		arpeggio := music.voice(1, frequencyOf(arpeggioNote), square(0.25)) * 0.14 * math.Max(0, 1-1.6*inStep)
		// A hi-hat on every off-beat eighth.
		hat := 0.0
		if step%2 == 1 && inStep < 0.3 {
			music.hat += 0.8 * ((music.random.Float64()*2 - 1) - music.hat)
			hat = (music.random.Float64()*2 - 1 - music.hat) * 0.08 * (1 - inStep/0.3)
		}
		samples[index] = float32(music.Volume * (bass + arpeggio + hat))
		music.frame++
	}
	return len(samples), nil
}

func (music *Music) voice(index int, frequency float64, shape wave) float64 {
	music.phases[index] = math.Mod(music.phases[index]+frequency/SampleRate, 1)
	return shape(music.phases[index])
}

// frequencyOf is a note's frequency, in semitones from the A above middle C (440 Hz).
func frequencyOf(semitones int) float64 { return 440 * math.Pow(2, float64(semitones)/12) }
