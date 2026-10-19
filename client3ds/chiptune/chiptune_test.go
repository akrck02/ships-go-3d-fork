package chiptune

import (
	"errors"
	"io"
	"math"
	"testing"
)

func everySound() map[string]*Sound {
	return map[string]*Sound{"laser": Laser(), "hit": Hit(), "explosion": Explosion(), "shield": Shield(), "move": Move(), "choose": Choose(), "back": Back()}
}

func TestEverySoundStaysWithinRangeAndEnds(t *testing.T) {
	for name, sound := range everySound() {
		buffer := make([]float32, 1000)
		loudest, total := 0.0, 0
		for {
			count, err := sound.Read(buffer)
			for _, sample := range buffer[:count] {
				loudest = math.Max(loudest, math.Abs(float64(sample)))
			}
			total += count
			if errors.Is(err, io.EOF) {
				break
			}
		}
		if loudest > 1 || loudest < 0.05 {
			t.Errorf("%s peaks at %v, want audible and within 1", name, loudest)
		}
		if seconds := float64(total) / SampleRate; seconds > 2 {
			t.Errorf("%s lasts %v s", name, seconds)
		}
	}
}

func TestASoundPlaysAgainAfterRewinding(t *testing.T) {
	laser := Laser()
	first := make([]float32, 100)
	laser.Read(first) //nolint:errcheck // the start of a sound longer than this
	laser.Rewind()    //nolint:errcheck // a buffer rewinds without fail
	again := make([]float32, 100)
	laser.Read(again) //nolint:errcheck // as above
	for index := range first {
		if first[index] != again[index] {
			t.Fatal("the rewound sound differs")
		}
	}
}

func TestTheMusicNeverEndsAndStaysWithinRange(t *testing.T) {
	music := NewMusic(1)
	buffer := make([]float32, SampleRate)
	for range 20 { // twenty seconds: past one loop of the progression
		count, err := music.Read(buffer)
		if err != nil || count != len(buffer) {
			t.Fatalf("got %d, %v", count, err)
		}
		for _, sample := range buffer {
			if math.Abs(float64(sample)) > 1 {
				t.Fatalf("a sample of %v", sample)
			}
		}
	}
}

func loudness(engine *Engine, seconds float64) float64 {
	buffer := make([]float32, int(seconds*SampleRate))
	engine.Read(buffer) //nolint:errcheck // the engine never fails
	total := 0.0
	for _, sample := range buffer {
		total += float64(sample) * float64(sample)
	}
	return math.Sqrt(total / float64(len(buffer)))
}

func TestTheEngineHumsLouderAtFullThrottleAndStaysInRange(t *testing.T) {
	engine := NewEngine(1)
	idle := loudness(engine, 0.5)
	engine.Throttle = 1
	loudness(engine, 1) // the pitch glides up first
	full := loudness(engine, 0.5)
	if idle < 0.02 || full <= idle*1.5 {
		t.Fatalf("idle %v, full %v: want an audible idle and a louder full throttle", idle, full)
	}
	buffer := make([]float32, SampleRate)
	count, err := engine.Read(buffer)
	if err != nil || count != len(buffer) {
		t.Fatalf("the engine stopped: %d samples, %v", count, err)
	}
	for _, sample := range buffer {
		if math.Abs(float64(sample)) > 1 {
			t.Fatalf("a sample of %v is out of range", sample)
		}
	}
}
