// Package turning shares out, frame by frame, which ships may show their new orientation. On the
// 3DS a ship that turns has its light worked out again on the processor (go-renderer's ctr bakes
// it), which costs about the same for every ship; with many players that would take the whole
// frame. So each frame only a few ships, the ones whose turn shows the most, are brought up to
// date, and the rest keep their last orientation a moment longer: a few frames late at most.
package turning

import (
	"math"

	"ships3d/client3ds/flight"
)

// Candidate is a ship that could be brought up to date.
type Candidate struct {
	Turned   float64 // radians between what is shown and where it faces now
	Distance float64 // from the camera, in world units
	Detail   bool    // it must change its level of detail too, which shows more
}

// nearest is the distance closer than which every ship counts as near: closer ships don't get
// more of the budget for it.
const nearest = 10

// detailWeight is how many radians of turning a change of detail is worth.
const detailWeight = 1

// Choose returns the indices of the candidates to bring up to date this frame, at most budget, the
// most visible change first: the larger the turn and the nearer the ship, the more it shows.
// The slice is reused by the next call.
func (chooser *Chooser) Choose(candidates []Candidate, budget int) []int {
	chooser.order = chooser.order[:0]
	for index, candidate := range candidates {
		if candidate.Turned > 0 || candidate.Detail {
			chooser.order = append(chooser.order, index)
		}
	}
	// An insertion sort: a few dozen ships, and unlike sort.Slice no closure to allocate.
	order := chooser.order
	for next := 1; next < len(order); next++ {
		for at := next; at > 0 && visibility(candidates[order[at]]) > visibility(candidates[order[at-1]]); at-- {
			order[at], order[at-1] = order[at-1], order[at]
		}
	}
	if len(chooser.order) > budget {
		chooser.order = chooser.order[:budget]
	}
	return chooser.order
}

// Chooser keeps Choose's list between frames, so choosing allocates nothing.
type Chooser struct {
	order []int
}

func visibility(candidate Candidate) float64 {
	change := candidate.Turned
	if candidate.Detail {
		change += detailWeight
	}
	return change / math.Max(candidate.Distance, nearest)
}

// Angle is how far apart two rotations are, in radians, from 0 to pi.
func Angle(first, second flight.Rotation) float64 {
	dot := math.Abs(first[0]*second[0] + first[1]*second[1] + first[2]*second[2] + first[3]*second[3])
	return 2 * math.Acos(math.Min(1, dot))
}
