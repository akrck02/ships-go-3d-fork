package impact

import "testing"

func TestTheGlowIsBrightestAtTheHitAndGoneAtItsReach(t *testing.T) {
	if Glow(0) != 1 || Glow(GlowReach) != 0 || Glow(2*GlowReach) != 0 {
		t.Fatalf("got %v %v %v", Glow(0), Glow(GlowReach), Glow(2*GlowReach))
	}
	if half := Glow(GlowReach / 2); half != 0.5 {
		t.Fatalf("half way %v, want 0.5", half)
	}
}

func TestEffectsFadeOut(t *testing.T) {
	if Fade(0, 1) != 1 || Fade(1, 1) != 0 || Fade(-1, 1) != 0 || Fade(0.5, 1) != 0.25 {
		t.Fatal("wrong fade")
	}
}

func TestTheShakeDiesAway(t *testing.T) {
	if x, y := Shake(ShakeSeconds); x != 0 || y != 0 {
		t.Fatalf("still shaking: %v %v", x, y)
	}
	x, _ := Shake(0.01)
	if x == 0 || x > ShakeReach {
		t.Fatalf("x %v at the start", x)
	}
}
