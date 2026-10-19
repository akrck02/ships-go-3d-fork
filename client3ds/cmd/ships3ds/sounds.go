//go:build nintendo3ds

package main

import (
	"time"

	sound "github.com/akrck02/go-sound"

	"ships3d/client3ds/chiptune"
)

// The game's sound, through go-sound (the console's DSP): chiptune effects and music worked out in
// code (package chiptune). Each effect has a few players taken in turn, so a burst of shots
// overlaps instead of cutting itself off. On a console without the DSP's firmware
// (sdmc:/3ds/dspfirm.cdc, dumped once with the DSP1 homebrew) the game plays on in silence.
const (
	soundLatency     = 60 * time.Millisecond // short enough for a shot to sound with it
	playersPerEffect = 4
	otherShotsVolume = 0.45 // others' shots, quieter than ours
)

// effect is one sound and its players.
type effect struct {
	players []*sound.Player
	next    int
}

// sounds are the game's effects and music; all of it nil when the console has no sound.
type sounds struct {
	context                       *sound.Context
	laser, hit, explosion, shield *effect
	move, choose, back            *effect
	music                         *sound.Player
	effectsVolume, musicVolume    float64
}

// stream gives go-sound a chiptune sound or the music.
type stream struct {
	source interface {
		Format() (sampleRate, channels int)
		Read(samples []float32) (int, error)
		Rewind() error
	}
}

func (played stream) Format() sound.Format {
	rate, channels := played.source.Format()
	return sound.Format{SampleRate: rate, Channels: channels}
}
func (played stream) Read(samples []float32) (int, error) { return played.source.Read(samples) }
func (played stream) Rewind() error                       { return played.source.Rewind() }

// newSounds opens the console's sound, or returns silent sounds and why there is none.
func newSounds() (*sounds, error) {
	all := &sounds{effectsVolume: 1, musicVolume: 1}
	context, err := sound.NewContext(sound.Options{Latency: soundLatency, Name: "Ships 3D"})
	if err != nil {
		return all, err
	}
	all.context = context
	for target, make := range map[**effect]func() *chiptune.Sound{
		&all.laser: chiptune.Laser, &all.hit: chiptune.Hit, &all.explosion: chiptune.Explosion, &all.shield: chiptune.Shield,
		&all.move: chiptune.Move, &all.choose: chiptune.Choose, &all.back: chiptune.Back,
	} {
		played := &effect{}
		made := make()
		for range playersPerEffect {
			player, err := context.NewPlayer(stream{made.Copy()})
			if err != nil {
				return all, err
			}
			played.players = append(played.players, player)
		}
		*target = played
	}
	music, err := context.NewPlayer(stream{chiptune.NewMusic(1)})
	if err != nil {
		return all, err
	}
	all.music = music
	return all, nil
}

// play starts an effect at a share of the effects' volume, on the player whose turn it is.
func (all *sounds) play(played *effect, volume float64) {
	if all.context == nil || played == nil {
		return
	}
	player := played.players[played.next]
	played.next = (played.next + 1) % len(played.players)
	player.Rewind() //nolint:errcheck // a buffer rewinds without fail
	player.SetVolume(volume * all.effectsVolume)
	player.Play()
}

// playMusic starts the music, looping, or stops it at no volume.
func (all *sounds) playMusic() {
	if all.music == nil {
		return
	}
	all.music.SetVolume(all.musicVolume * musicLevel)
	if all.musicVolume == 0 {
		all.music.Pause()
		return
	}
	if !all.music.IsPlaying() {
		all.music.SetLoop(true) //nolint:errcheck // the music rewinds without fail
		all.music.Play()
	}
}

// musicLevel keeps the music under the effects.
const musicLevel = 0.55

// setVolumes takes the settings' volumes, from 0 to 1.
func (all *sounds) setVolumes(effects, music float64) {
	all.effectsVolume, all.musicVolume = effects, music
	all.playMusic()
}

// musicSeconds is how much of the music has been heard, for the log: it stops growing when the DSP
// stops being fed.
func (all *sounds) musicSeconds() float64 {
	if all.music == nil {
		return 0
	}
	return all.music.Position().Seconds()
}
