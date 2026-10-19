// Package impact is what a crash into the world's edge looks like: how bright the edge glows
// around where the ship hit it, how sparks fly, and how the camera shakes, each fading out.
package impact

import "math"

// How long each effect lasts, in seconds, and how far the glow spreads, in world units.
const (
	GlowSeconds  = 1.0
	SparkSeconds = 0.7
	ShakeSeconds = 0.45
	GlowReach    = 260
	// WaveSeconds is how long the force field's ring takes to spread to WaveReach.
	WaveSeconds = 0.9
	WaveReach   = 320
	// FlashSeconds is how long the struck face of the force field stays lit.
	FlashSeconds = 0.6
	// ShakeReach is how far the camera moves at a crash's first moment, in world units.
	ShakeReach = 0.9
)

// Glow is how bright the edge is at distance from the hit, from 1 there to 0 at GlowReach, along a
// smooth curve, so the glow has no hard rim.
func Glow(distance float64) float64 {
	share := 1 - math.Min(distance/GlowReach, 1)
	return share * share * (3 - 2*share) // smoothstep
}

// Fade is how much of an effect of duration remains after seconds: 1 at once, 0 when over,
// easing out.
func Fade(seconds, duration float64) float64 {
	if seconds < 0 || seconds >= duration {
		return 0
	}
	left := 1 - seconds/duration
	return left * left
}

// Shake is the camera's offset seconds after a crash: a quick wobble on two axes that dies away.
func Shake(seconds float64) (x, y float64) {
	strength := Fade(seconds, ShakeSeconds) * ShakeReach
	// Two frequencies that don't share a beat, so the wobble doesn't look like a circle.
	const across, upAndDown = 47, 61
	return strength * math.Sin(seconds*across), strength * math.Sin(seconds*upAndDown+1)
}

// WaveRadius is how far the force field's ring has spread seconds after a crash: fast at first,
// slowing as it goes, as a ripple does.
func WaveRadius(seconds float64) float64 {
	share := math.Min(math.Max(seconds/WaveSeconds, 0), 1)
	return WaveReach * (1 - (1-share)*(1-share))
}
