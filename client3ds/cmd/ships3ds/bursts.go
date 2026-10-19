//go:build nintendo3ds

package main

import (
	"math/rand/v2"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"

	"ships3d/client3ds/flight"
	"ships3d/client3ds/game"
	"ships3d/client3ds/impact"
	"ships3d/client3ds/protocol"
	"ships3d/client3ds/ship"
)

// Bursts: a bullet's strike flashes on the ship it hit, and a destroyed ship blows up in a
// fireball that swells and fades, throwing debris. Each burst is a sphere with a material of its
// own, so each fades alone; there are a few, used in turn.
const (
	burstCount       = 12
	debrisCount      = 40
	debrisPerBlast   = 14
	hitSeconds       = 0.25
	hitSize          = 3.5 // radius, world units, at its largest
	blastSeconds     = 1.1
	blastSize        = 16
	debrisSeconds    = 1.4
	debrisSpeed      = 40
	debrisSize       = 1.1
	burstSeed        = 9
	ownHitShakeShare = 0.5  // a hit on us shakes the camera half as hard as a crash
	ownBlastShare    = 0.35 // our own blast, seen from just behind
)

var (
	hitColour    = graphics.Vec4{1, 0.95, 0.7, 1}
	blastColour  = graphics.Vec4{1, 0.45, 0.12, 1}
	debrisColour = [3]float32{1, 0.6, 0.25}
)

// burst is one flash or fireball.
type burst struct {
	node     *scene.Node
	at       float64
	seconds  float64
	size     float64
	colour   graphics.Vec4
	position flight.Vector
}

// piece is one piece of debris.
type piece struct {
	node     *scene.Node
	at       float64
	position flight.Vector
	velocity flight.Vector
}

// bursts are the flashes, fireballs and debris in flight.
type bursts struct {
	flashes []burst
	debris  []piece
	next    int // the flash to use next
	nextBit int // the piece of debris to use next
	random  *rand.Rand
}

func newBursts(space *scene.Scene) *bursts {
	all := &bursts{random: rand.New(rand.NewPCG(burstSeed, burstSeed))}
	ball := ship.Build([]protocol.Layer{{Visible: true, Parts: []protocol.Part{{
		Type: "sphere", Scale: flight.Vector{2, 2, 2}, Color: "#ffffff"}}}}, ship.Coarse)
	ball.Material = unlit()
	for range burstCount {
		node := &scene.Node{Name: "burst", Mesh: withOwnMaterial(ball), Hidden: true, Moving: true}
		all.flashes = append(all.flashes, burst{node: node})
		space.Nodes = append(space.Nodes, node)
	}
	bit := box("debris", debrisSize, debrisSize, debrisSize, debrisColour)
	for range debrisCount {
		node := &scene.Node{Name: "debris", Mesh: withOwnMaterial(bit), Hidden: true, Moving: true}
		all.debris = append(all.debris, piece{node: node})
		space.Nodes = append(space.Nodes, node)
	}
	return all
}

// start shows a game's event. A strike on our own ship shows as the camera's shake and the
// radar's red edge instead: a flash around it would hide it, right in front of the camera.
func (all *bursts) start(event game.Event) {
	if event.Kind == game.ShipHit && event.Own {
		return
	}
	flash := &all.flashes[all.next]
	all.next = (all.next + 1) % len(all.flashes)
	flash.at, flash.position = event.At, event.Position
	switch event.Kind {
	case game.ShipHit:
		flash.seconds, flash.size, flash.colour = hitSeconds, hitSize, hitColour
	case game.ShipDestroyed:
		flash.seconds, flash.size, flash.colour = blastSeconds, blastSize, blastColour
		if event.Own {
			flash.size = blastSize * ownBlastShare // the camera is close: a full one would swallow it
		}
		all.throwDebris(event)
	}
}

func (all *bursts) throwDebris(event game.Event) {
	for range debrisPerBlast {
		bit := &all.debris[all.nextBit]
		all.nextBit = (all.nextBit + 1) % len(all.debris)
		direction := flight.Vector{all.random.Float64()*2 - 1, all.random.Float64()*2 - 1, all.random.Float64()*2 - 1}
		if direction.Length() == 0 {
			direction = flight.Forward
		}
		bit.at, bit.position = event.At, event.Position
		bit.velocity = direction.Times(debrisSpeed * (0.3 + 0.7*all.random.Float64()) / direction.Length())
	}
}

// show moves every burst on to now.
func (all *bursts) show(now, seconds float64) {
	for index := range all.flashes {
		flash := &all.flashes[index]
		age := now - flash.at
		fade := impact.Fade(age, flash.seconds)
		flash.node.Hidden = flash.seconds == 0 || fade == 0
		if flash.node.Hidden {
			continue
		}
		// Swells fast, then holds while it fades.
		grown := min(1, age/(flash.seconds*0.35))
		scale := float32(flash.size * (0.3 + 0.7*grown))
		flash.node.Transform = graphics.Translate(vectorOf(flash.position, 0)).Multiply(graphics.Scale(graphics.Vec4{scale, scale, scale, 0}))
		shade := float32(fade)
		flash.node.Mesh.Material.BaseColor = graphics.Vec4{flash.colour[0] * shade, flash.colour[1] * shade, flash.colour[2] * shade, 1}
	}
	for index := range all.debris {
		bit := &all.debris[index]
		fade := impact.Fade(now-bit.at, debrisSeconds)
		bit.node.Hidden = bit.at == 0 || fade == 0
		if bit.node.Hidden {
			continue
		}
		bit.position = bit.position.Add(bit.velocity.Times(seconds))
		bit.node.Transform = graphics.Translate(vectorOf(bit.position, 0))
		shade := float32(fade)
		bit.node.Mesh.Material.BaseColor = graphics.Vec4{shade, shade, shade, 1}
	}
}
