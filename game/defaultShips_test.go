package game

import "testing"

func TestEveryDefaultShipIsAValidDesign(t *testing.T) {
	for _, info := range DefaultShips() {
		project, found := DefaultShip(info.Id)
		if !found {
			t.Fatalf("%s is listed but can't be found by its id", info.Id)
		}
		if err := project.Validate(); err != nil {
			t.Errorf("%s: %v", info.Name, err)
		}
		if project.BoundingRadius() <= 0 {
			t.Errorf("%s has no size, so it can't be flown", info.Name)
		}
	}
}

func TestTheDefaultShipsHaveTheirIds(t *testing.T) {
	want := []string{"default:arrow", "default:hammer", "default:saucer", "default:kestrel", "default:mule", "default:lancer", "default:bulwark", "default:wisp"}
	ships := DefaultShips()
	if len(ships) != len(want) {
		t.Fatalf("%d ships, want %d", len(ships), len(want))
	}
	for index, ship := range ships {
		if ship.Id != want[index] {
			t.Errorf("ship %d is %s, want %s", index, ship.Id, want[index])
		}
	}
}
