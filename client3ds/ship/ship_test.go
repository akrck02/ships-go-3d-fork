package ship

import (
	"math"
	"testing"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/protocol"
)

var everyType = []string{partBox, partSphere, partCylinder, partCone, partWedge, partTorus, partDome, partCapsule, partPyramid, partTube, partPrism, partFrustum}

// slack is how far a corner may stray past the cube from rounding.
const slack = 1e-9

func TestEachPrimitiveFitsTheUnitCube(t *testing.T) {
	for _, kind := range everyType {
		triangles := primitive(kind)
		if len(triangles) == 0 {
			t.Errorf("%s has no triangles", kind)
		}
		var low, high point
		for _, corners := range triangles {
			for _, corner := range corners {
				for axis := range 3 {
					low[axis], high[axis] = min(low[axis], corner[axis]), max(high[axis], corner[axis])
				}
			}
		}
		for axis := range 3 {
			if low[axis] < -0.5-slack || high[axis] > 0.5+slack {
				t.Errorf("%s spans %v to %v, outside the unit cube", kind, low, high)
			}
		}
	}
}

func TestEachPrimitiveFillsItsHeight(t *testing.T) {
	for _, kind := range everyType {
		if kind == partTorus {
			continue // it lies in XY, a ring around Z
		}
		low, high := math.Inf(1), math.Inf(-1)
		for _, corners := range primitive(kind) {
			for _, corner := range corners {
				low, high = min(low, corner[1]), max(high, corner[1])
			}
		}
		if math.Abs(low+0.5) > slack || math.Abs(high-0.5) > slack {
			t.Errorf("%s goes from %v to %v in height, want -0.5 to 0.5", kind, low, high)
		}
	}
}

func boxPart(position flight.Vector, mirror bool) protocol.Part {
	return protocol.Part{Type: partBox, Position: position, Scale: flight.Vector{1, 1, 1}, Color: "#ffffff", Mirror: mirror}
}

func TestAMirroredPartIsReflectedAcrossX(t *testing.T) {
	mesh := Build([]protocol.Layer{{Visible: true, Parts: []protocol.Part{boxPart(flight.Vector{3, 0, 0}, true)}}})
	lowest, highest := float32(math.Inf(1)), float32(math.Inf(-1))
	for index := 0; index < len(mesh.Positions); index += 3 {
		lowest, highest = min(lowest, mesh.Positions[index]), max(highest, mesh.Positions[index])
	}
	if lowest != -3.5 || highest != 3.5 {
		t.Fatalf("x from %v to %v, want a box at each side", lowest, highest)
	}
}

func TestHiddenLayersAreNotBuilt(t *testing.T) {
	mesh := Build([]protocol.Layer{{Visible: false, Parts: []protocol.Part{boxPart(flight.Vector{}, false)}}})
	if mesh.VertexCount() != 0 {
		t.Fatalf("%d vertices from a hidden layer", mesh.VertexCount())
	}
}

func TestTheRadiusIsTheServersBoundingRadius(t *testing.T) {
	layers := []protocol.Layer{{Visible: true, Parts: []protocol.Part{{Position: flight.Vector{3, 4, 0}, Scale: flight.Vector{2, 0, 0}}}}}
	if got := Radius(layers); got != 6 {
		t.Fatalf("got %v, want 5 + 2/2", got)
	}
}

func TestColoursAreReadAsSRGB(t *testing.T) {
	if got := linearColour("#ff0080"); got[0] != 1 || got[1] != 0 || math.Abs(float64(got[2])-0.2158605) > 1e-5 {
		t.Fatalf("got %v", got)
	}
	if got := linearColour("not a colour"); got == ([3]float32{}) {
		t.Fatal("a bad colour came out black, want the fallback grey")
	}
}
