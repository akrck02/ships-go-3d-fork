package radar

import "testing"

func TestAShipShowsOnlyOnceTheSweepPassesIt(t *testing.T) {
	scanner := NewScanner()
	right := map[string]Blip{"them": {Right: 0.5}} // a quarter turn from ahead
	scanner.Update(0, right)
	scanner.Update(ScanSeconds*0.2, right)
	if _, seen := scanner.Echoes["them"]; seen {
		t.Fatal("seen before the sweep reached it")
	}
	scanner.Update(ScanSeconds*0.3, right)
	if _, seen := scanner.Echoes["them"]; !seen {
		t.Fatal("not seen once the sweep passed it")
	}
}

func TestAnEchoStaysWhereItWasSeen(t *testing.T) {
	scanner := NewScanner()
	scanner.Update(0, map[string]Blip{"them": {Right: 0.5}})
	scanner.Update(ScanSeconds*0.3, map[string]Blip{"them": {Right: 0.5}})
	scanner.Update(ScanSeconds*0.35, map[string]Blip{"them": {Right: 0.5, Ahead: -0.9}}) // moved behind
	if echo := scanner.Echoes["them"]; echo.Blip.Ahead != 0 {
		t.Fatalf("the echo moved to %+v before the sweep came round", echo.Blip)
	}
}

func TestTheSweepWrapsRoundStraightAhead(t *testing.T) {
	scanner := NewScanner()
	ahead := map[string]Blip{"them": {Ahead: 0.5, Right: -0.01}} // just before a whole turn
	scanner.Update(ScanSeconds*0.9, ahead)
	scanner.Update(ScanSeconds*1.05, ahead)
	if _, seen := scanner.Echoes["them"]; !seen {
		t.Fatal("a ship just left of ahead was missed as the sweep went round")
	}
}

func TestEchoesFadeOverATurn(t *testing.T) {
	echo := Echo{SeenAt: 10}
	if echo.Brightness(10) != 1 || echo.Brightness(10+ScanSeconds/2.0) != 0.5 || echo.Brightness(10+2*ScanSeconds) != 0 {
		t.Fatal("wrong fade")
	}
}

func TestShipsThatLeftAreForgotten(t *testing.T) {
	scanner := NewScanner()
	scanner.Echoes["gone"] = Echo{}
	scanner.Update(1, map[string]Blip{})
	if len(scanner.Echoes) != 0 {
		t.Fatal("kept a ship that left")
	}
}
