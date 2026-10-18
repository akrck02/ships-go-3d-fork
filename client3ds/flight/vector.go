// Package flight is the ships' flight model, the same as the browser's (ships-vue-3d,
// src/game/flight.ts): arcade flight with six degrees of freedom, where a ship turns at fixed
// rates in its own frame and always moves along its nose. It also holds the small vector and
// rotation arithmetic it needs, in float64 like the server's.
package flight

import "math"

// Vector is a point or a direction in the world: [x, y, z], as the protocol sends it.
type Vector [3]float64

// Add returns vector + other.
func (vector Vector) Add(other Vector) Vector {
	return Vector{vector[0] + other[0], vector[1] + other[1], vector[2] + other[2]}
}

// Subtract returns vector - other.
func (vector Vector) Subtract(other Vector) Vector {
	return Vector{vector[0] - other[0], vector[1] - other[1], vector[2] - other[2]}
}

// Times returns the vector multiplied by factor.
func (vector Vector) Times(factor float64) Vector {
	return Vector{vector[0] * factor, vector[1] * factor, vector[2] * factor}
}

// Dot is the dot product.
func (vector Vector) Dot(other Vector) float64 {
	return vector[0]*other[0] + vector[1]*other[1] + vector[2]*other[2]
}

// Length is the vector's length.
func (vector Vector) Length() float64 { return math.Sqrt(vector.Dot(vector)) }

// Toward moves from vector toward target by share (0 stays, 1 arrives).
func (vector Vector) Toward(target Vector, share float64) Vector {
	return vector.Add(target.Subtract(vector).Times(share))
}

// Rotation is a rotation as a unit quaternion [x, y, z, w], three.js's order, as the protocol
// sends it.
type Rotation [4]float64

// NoRotation leaves everything as it is.
var NoRotation = Rotation{0, 0, 0, 1}

// Then is the rotation of turning by rotation and then by next in rotation's own frame (three.js's
// quaternion.multiply).
func (rotation Rotation) Then(next Rotation) Rotation {
	ax, ay, az, aw := rotation[0], rotation[1], rotation[2], rotation[3]
	bx, by, bz, bw := next[0], next[1], next[2], next[3]
	return Rotation{
		ax*bw + aw*bx + ay*bz - az*by,
		ay*bw + aw*by + az*bx - ax*bz,
		az*bw + aw*bz + ax*by - ay*bx,
		aw*bw - ax*bx - ay*by - az*bz,
	}
}

// Normalized is the rotation with length 1, which rounding slowly takes it away from.
func (rotation Rotation) Normalized() Rotation {
	length := math.Sqrt(rotation[0]*rotation[0] + rotation[1]*rotation[1] + rotation[2]*rotation[2] + rotation[3]*rotation[3])
	if length == 0 {
		return NoRotation
	}
	return Rotation{rotation[0] / length, rotation[1] / length, rotation[2] / length, rotation[3] / length}
}

// Apply turns a vector by the rotation.
func (rotation Rotation) Apply(vector Vector) Vector {
	x, y, z, w := rotation[0], rotation[1], rotation[2], rotation[3]
	// The quaternion times the vector, then times the quaternion's inverse, in two steps.
	ix := w*vector[0] + y*vector[2] - z*vector[1]
	iy := w*vector[1] + z*vector[0] - x*vector[2]
	iz := w*vector[2] + x*vector[1] - y*vector[0]
	iw := -x*vector[0] - y*vector[1] - z*vector[2]
	return Vector{
		ix*w + iw*-x + iy*-z - iz*-y,
		iy*w + iw*-y + iz*-x - ix*-z,
		iz*w + iw*-z + ix*-y - iy*-x,
	}
}

// Toward turns from rotation toward target by share, along the shorter way (three.js's slerp).
func (rotation Rotation) Toward(target Rotation, share float64) Rotation {
	cosine := rotation[0]*target[0] + rotation[1]*target[1] + rotation[2]*target[2] + rotation[3]*target[3]
	if cosine < 0 {
		target = Rotation{-target[0], -target[1], -target[2], -target[3]}
		cosine = -cosine
	}
	// Close rotations are blended straight, where the sine below would divide by almost nothing.
	const straightBlendCosine = 0.9995
	if cosine > straightBlendCosine {
		var blended Rotation
		for index := range blended {
			blended[index] = rotation[index] + (target[index]-rotation[index])*share
		}
		return blended.Normalized()
	}
	angle := math.Acos(cosine)
	sine := math.Sin(angle)
	fromWeight, toWeight := math.Sin((1-share)*angle)/sine, math.Sin(share*angle)/sine
	var blended Rotation
	for index := range blended {
		blended[index] = rotation[index]*fromWeight + target[index]*toWeight
	}
	return blended
}

// FromEulerXYZ is the rotation of Euler angles in radians, turned about X, then Y, then Z in the
// moving frame: three.js's default order, which the ship designs use.
func FromEulerXYZ(x, y, z float64) Rotation {
	cosineX, sineX := math.Cos(x/2), math.Sin(x/2)
	cosineY, sineY := math.Cos(y/2), math.Sin(y/2)
	cosineZ, sineZ := math.Cos(z/2), math.Sin(z/2)
	return Rotation{
		sineX*cosineY*cosineZ + cosineX*sineY*sineZ,
		cosineX*sineY*cosineZ - sineX*cosineY*sineZ,
		cosineX*cosineY*sineZ + sineX*sineY*cosineZ,
		cosineX*cosineY*cosineZ - sineX*sineY*sineZ,
	}
}

// Inverse is the rotation that undoes this one.
func (rotation Rotation) Inverse() Rotation {
	return Rotation{-rotation[0], -rotation[1], -rotation[2], rotation[3]}
}
