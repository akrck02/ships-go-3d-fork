// Package profile is what the game remembers between runs, on the console's SD card: the server,
// the logged-in session, the last ship flown and the player's settings. It only turns them into
// bytes and back; the program decides where they are kept.
package profile

import "encoding/json"

// Settings are the player's choices in the Settings tab.
type Settings struct {
	InvertPitch bool    `json:"invertPitch"` // up on the circle pad raises the nose, instead of a plane's way
	Precision   float32 `json:"precision"`   // 0 to 1: how much softer the circle pad is near its centre
	Gyroscope   bool    `json:"gyroscope"`   // tilting the console steers too
	FlatRadar   bool    `json:"flatRadar"`   // the touch screen's radar seen straight from above, as a map
	Effects     float32 `json:"effects"`     // the sound effects' volume, 0 to 1
	Music       float32 `json:"music"`       // the music's volume, 0 to 1
}

// Profile is everything remembered.
type Profile struct {
	Server   string   `json:"server"`
	Email    string   `json:"email"`    // the last one typed, offered again
	Token    string   `json:"token"`    // the session, kept instead of the password
	Username string   `json:"username"` // the session's player
	Ship     string   `json:"ship"`     // the id of the last ship flown
	Settings Settings `json:"settings"`
}

// The settings before the player changes them.
const (
	DefaultPrecision = 0.5
	DefaultEffects   = 0.8
	DefaultMusic     = 0.6
)

// New is the profile of a first run, starting with server.
func New(server string) Profile {
	return Profile{Server: server, Settings: Settings{Precision: DefaultPrecision, Effects: DefaultEffects, Music: DefaultMusic}}
}

// Read understands saved bytes, starting from a first run's profile with server: a field missing,
// or bytes that can't be read at all, keep their first-run value.
func Read(saved []byte, server string) Profile {
	read := New(server)
	if json.Unmarshal(saved, &read) != nil {
		return New(server) // unreadable, perhaps cut short: start afresh rather than half-read
	}
	if read.Server == "" {
		read.Server = server
	}
	return read
}

// Bytes is the profile ready to save.
func (profile Profile) Bytes() []byte {
	bytes, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return nil // only unencodable numbers fail, and a profile holds none
	}
	return bytes
}

// LoggedIn tells whether a session is remembered.
func (profile Profile) LoggedIn() bool { return profile.Token != "" }
