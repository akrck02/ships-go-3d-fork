package profile

import "testing"

const realServer = "https://example.org:3000"

func TestAProfileComesBackAsItWasSaved(t *testing.T) {
	saved := Profile{Server: "192.168.1.20:3000", Email: "pilot@example.org", Token: "abc", Username: "pilot", Ship: "default:wisp",
		Settings: Settings{InvertPitch: true, Precision: 0.2, Gyroscope: true, FlatRadar: true, Effects: 0.3, Music: 0}}
	if got := Read(saved.Bytes(), realServer); got != saved {
		t.Fatalf("got %+v, want %+v", got, saved)
	}
}

func TestAFirstRunStartsWithTheRealServer(t *testing.T) {
	got := Read(nil, realServer)
	if got.Server != realServer || got.LoggedIn() || got.Settings.Precision != DefaultPrecision {
		t.Fatalf("got %+v", got)
	}
}

func TestMissingFieldsKeepTheirFirstRunValues(t *testing.T) {
	got := Read([]byte(`{"email":"pilot@example.org"}`), realServer)
	if got.Server != realServer || got.Email != "pilot@example.org" || got.Settings.Precision != DefaultPrecision || got.Settings.Music != DefaultMusic {
		t.Fatalf("got %+v", got)
	}
}

func TestABrokenFileStartsAfresh(t *testing.T) {
	if got := Read([]byte(`{"server":"x", "tok`), realServer); got != New(realServer) {
		t.Fatalf("got %+v", got)
	}
}
