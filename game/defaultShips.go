package game

import (
	"math"
	"strings"

	"ships3d/models"
)

// DefaultShipPrefix marks a built-in ship id, as opposed to the ObjectID of a
// player's own project. Guests can only fly built-in ships.
const DefaultShipPrefix = "default:"

// Ships face +Z (their nose) with +Y up. The editor shows the same axes.
var defaultShips = []models.Project3d{
	{
		Name: "Arrow",
		Layers: []models.Layer3d{{Name: "Hull", Visible: true, Parts: []models.Part{
			part(models.PartBox, v(0, 0, 0), v(0, 0, 0), v(1.2, 0.8, 4), "#9aa7b8", false),
			part(models.PartCone, v(0, 0, 2.6), v(math.Pi/2, 0, 0), v(1.2, 1.2, 1.2), "#9aa7b8", false),
			part(models.PartSphere, v(0, 0.5, 0.6), v(0, 0, 0), v(0.7, 0.5, 1.2), "#4fc3f7", false),
			part(models.PartWedge, v(1.6, 0, -0.6), v(0, 0, 0), v(2.2, 0.2, 2), "#e05a47", true),
			part(models.PartCylinder, v(0.5, 0, -2.2), v(math.Pi/2, 0, 0), v(0.5, 0.6, 0.5), "#555b66", true),
		}}},
	},
	{
		Name: "Hammer",
		Layers: []models.Layer3d{{Name: "Hull", Visible: true, Parts: []models.Part{
			part(models.PartBox, v(0, 0, 0), v(0, 0, 0), v(2.5, 1.4, 3.5), "#6d7b4d", false),
			part(models.PartBox, v(0, 0, 2.2), v(0, 0, 0), v(3.5, 1, 0.8), "#8c9a63", false),
			part(models.PartCylinder, v(1.4, 0, 2.8), v(math.Pi/2, 0, 0), v(0.3, 1.6, 0.3), "#333333", true),
			part(models.PartSphere, v(0, 0.8, 0.5), v(0, 0, 0), v(1, 0.6, 1.2), "#ffd54f", false),
			part(models.PartCylinder, v(0.8, 0, -2), v(math.Pi/2, 0, 0), v(0.8, 0.8, 0.8), "#444444", true),
		}}},
	},
	{
		Name: "Saucer",
		Layers: []models.Layer3d{{Name: "Hull", Visible: true, Parts: []models.Part{
			part(models.PartSphere, v(0, 0, 0), v(0, 0, 0), v(4, 0.9, 4), "#b0bec5", false),
			part(models.PartTorus, v(0, 0, 0), v(math.Pi/2, 0, 0), v(5, 5, 1.5), "#ff7043", false),
			part(models.PartSphere, v(0, 0.5, 0), v(0, 0, 0), v(1.6, 1.2, 1.6), "#80deea", false),
			part(models.PartCone, v(0, 0, 2.4), v(math.Pi/2, 0, 0), v(0.6, 0.8, 0.6), "#ff7043", false),
		}}},
	},
	{
		Name: "Kestrel",
		Layers: []models.Layer3d{
			{Name: "Body", Visible: true, Parts: []models.Part{
				part(models.PartCapsule, v(0, 0, 0), v(math.Pi/2, 0, 0), v(1.4, 7, 1.2), "#b8c2cc", false),
				part(models.PartCone, v(0, 0, 4.4), v(math.Pi/2, 0, 0), v(1.1, 1.8, 0.9), "#b8c2cc", false),
				part(models.PartCylinder, v(0, 0, 5.6), v(math.Pi/2, 0, 0), v(0.12, 1.2, 0.12), "#3a4350", false),
				part(models.PartDome, v(0, 0.55, 1.4), v(0, 0, 0), v(0.9, 0.7, 1.9), "#4fc3f7", false),
				part(models.PartBox, v(0, 0.62, -0.1), v(0, 0, 0), v(0.5, 0.25, 1), "#7d8894", false),
				part(models.PartBox, v(0, -0.5, 0.5), v(0, 0, 0), v(0.7, 0.3, 3.2), "#3a4350", false),
				part(models.PartBox, v(0, 0.66, 1.4), v(0, 0, 0), v(0.08, 0.06, 1.6), "#e05a47", false),
			}},
			{Name: "Wings and fins", Visible: true, Parts: []models.Part{
				part(models.PartBox, v(0.85, -0.1, 0.6), v(0, 0, 0), v(0.5, 0.6, 1.6), "#7d8894", true),
				part(models.PartCylinder, v(0.85, -0.1, 1.5), v(math.Pi/2, 0, 0), v(0.45, 0.2, 0.55), "#3a4350", true),
				part(models.PartWedge, v(2.4, -0.15, -1), v(0, -0.25, -0.06), v(3.2, 0.18, 2.8), "#b8c2cc", true),
				part(models.PartBox, v(2.4, -0.04, -0.6), v(0, -0.25, -0.06), v(2.6, 0.04, 0.12), "#e05a47", true),
				part(models.PartCylinder, v(4, -0.25, 0), v(math.Pi/2, 0, 0), v(0.22, 3.2, 0.22), "#7d8894", true),
				part(models.PartCylinder, v(4, -0.25, 1.8), v(math.Pi/2, 0, 0), v(0.09, 0.8, 0.09), "#3a4350", true),
				part(models.PartSphere, v(4, -0.25, -1.6), v(0, 0, 0), v(0.3, 0.3, 0.3), "#3a4350", true),
				part(models.PartWedge, v(0.7, 0.9, -2.6), v(0, 0, -0.35), v(0.15, 1.6, 1.6), "#b8c2cc", true),
				part(models.PartBox, v(0.95, 1.55, -2.9), v(0, 0, -0.35), v(0.17, 0.12, 0.5), "#e05a47", true),
			}},
			{Name: "Engines", Visible: true, Parts: []models.Part{
				part(models.PartCylinder, v(0.75, 0, -3.3), v(math.Pi/2, 0, 0), v(0.7, 1.6, 0.7), "#7d8894", true),
				part(models.PartTube, v(0.75, 0, -4.2), v(math.Pi/2, 0, 0), v(0.8, 0.5, 0.8), "#3a4350", true),
				part(models.PartTorus, v(0.75, 0, -3.8), v(0, 0, 0), v(1.05, 1.05, 0.5), "#3a4350", true),
				part(models.PartSphere, v(0.75, 0, -4.3), v(0, 0, 0), v(0.5, 0.5, 0.2), "#ffb74d", true),
			}},
		},
	},
	{
		Name: "Mule",
		Layers: []models.Layer3d{
			{Name: "Cabin", Visible: true, Parts: []models.Part{
				part(models.PartBox, v(0, 0.2, 5), v(0, 0, 0), v(2.2, 1.8, 2), "#ffd54f", false),
				part(models.PartWedge, v(0, 0.2, 6.6), v(0, math.Pi, 0), v(2.2, 1.8, 1.2), "#ffd54f", false),
				part(models.PartBox, v(0, 0.75, 6.1), v(-0.55, 0, 0), v(1.8, 0.05, 0.9), "#4fc3f7", false),
				part(models.PartCylinder, v(0.6, 1.4, 4.6), v(0, 0, 0), v(0.06, 0.8, 0.06), "#555b66", true),
				part(models.PartSphere, v(0.6, 1.85, 4.6), v(0, 0, 0), v(0.14, 0.14, 0.14), "#ff8a65", true),
				part(models.PartBox, v(1.15, -0.3, 6.2), v(0, 0, 0), v(0.1, 0.25, 0.25), "#ff8a65", true),
			}},
			{Name: "Cargo", Visible: true, Parts: []models.Part{
				part(models.PartBox, v(0, -0.9, -0.5), v(0, 0, 0), v(0.5, 0.4, 11), "#555b66", false),
				part(models.PartBox, v(0.65, 0.1, 2.6), v(0, 0, 0), v(1.2, 1.6, 2), "#8c9a63", true),
				part(models.PartBox, v(0, 1.05, 2.6), v(0, 0, 0), v(2.5, 0.12, 0.2), "#555b66", false),
				part(models.PartBox, v(1.27, 0.1, 2.6), v(0, 0, 0), v(0.04, 1.3, 1.6), "#2e333b", true),
				part(models.PartBox, v(0.65, 0.1, 0.4), v(0, 0, 0), v(1.2, 1.6, 2), "#6d7b4d", true),
				part(models.PartBox, v(0, 1.05, 0.4), v(0, 0, 0), v(2.5, 0.12, 0.2), "#555b66", false),
				part(models.PartBox, v(1.27, 0.1, 0.4), v(0, 0, 0), v(0.04, 1.3, 1.6), "#2e333b", true),
				part(models.PartBox, v(0.65, 0.1, -1.8), v(0, 0, 0), v(1.2, 1.6, 2), "#8c9a63", true),
				part(models.PartBox, v(0, 1.05, -1.8), v(0, 0, 0), v(2.5, 0.12, 0.2), "#555b66", false),
				part(models.PartBox, v(1.27, 0.1, -1.8), v(0, 0, 0), v(0.04, 1.3, 1.6), "#2e333b", true),
				part(models.PartBox, v(0.65, 0.1, -4), v(0, 0, 0), v(1.2, 1.6, 2), "#6d7b4d", true),
				part(models.PartBox, v(0, 1.05, -4), v(0, 0, 0), v(2.5, 0.12, 0.2), "#555b66", false),
				part(models.PartBox, v(1.27, 0.1, -4), v(0, 0, 0), v(0.04, 1.3, 1.6), "#2e333b", true),
			}},
			{Name: "Engines", Visible: true, Parts: []models.Part{
				part(models.PartBox, v(0, 0.1, -6.3), v(0, 0, 0), v(2.6, 1.8, 1.2), "#555b66", false),
				part(models.PartCylinder, v(1.9, 0, -6), v(math.Pi/2, 0, 0), v(0.9, 2.6, 0.9), "#2e333b", true),
				part(models.PartFrustum, v(1.9, 0, -7.6), v(-math.Pi/2, 0, 0), v(1.1, 0.8, 1.1), "#555b66", true),
				part(models.PartSphere, v(1.9, 0, -8), v(0, 0, 0), v(0.7, 0.7, 0.25), "#ff8a65", true),
				part(models.PartBox, v(1.4, 0, -6), v(0, 0, 0), v(0.6, 0.3, 0.6), "#555b66", true),
				part(models.PartPrism, v(0, 1.2, -6.3), v(0, 0, 0), v(0.6, 0.6, 0.6), "#2e333b", false),
			}},
		},
	},
	{
		Name: "Lancer",
		Layers: []models.Layer3d{
			{Name: "Body", Visible: true, Parts: []models.Part{
				part(models.PartCone, v(0, 0, 5.5), v(math.Pi/2, 0, 0), v(0.7, 5, 0.6), "#e8edf2", false),
				part(models.PartCylinder, v(0, 0, 1.5), v(math.Pi/2, 0, 0), v(0.7, 3, 0.6), "#e8edf2", false),
				part(models.PartFrustum, v(0, 0, -1.2), v(-math.Pi/2, 0, 0), v(1.2, 2.4, 1), "#3d6fb6", false),
				part(models.PartCapsule, v(0, 0.4, 2.2), v(math.Pi/2, 0, 0), v(0.45, 1.8, 0.4), "#90caf9", false),
				part(models.PartBox, v(0, 0.38, 4), v(0, 0, 0), v(0.05, 0.05, 2.5), "#3d6fb6", false),
			}},
			{Name: "Fins", Visible: true, Parts: []models.Part{
				part(models.PartWedge, v(0.9, 0, 3.4), v(0, 0, -0.1), v(1.2, 0.08, 1), "#3d6fb6", true),
				part(models.PartWedge, v(1.8, 0, -1.6), v(0, 0, -0.1), v(2.8, 0.1, 2.4), "#e8edf2", true),
				part(models.PartBox, v(3.1, -0.1, -1.6), v(0, 0, 0), v(0.12, 0.6, 1.6), "#3d6fb6", true),
				part(models.PartWedge, v(0, 1, -1.8), v(0, 0, 0), v(0.1, 1.6, 2), "#3d6fb6", false),
				part(models.PartWedge, v(0, -0.8, -1.8), v(0, 0, math.Pi), v(0.1, 1, 1.6), "#3d6fb6", false),
			}},
			{Name: "Drive", Visible: true, Parts: []models.Part{
				part(models.PartTorus, v(0, 0, -3), v(0, 0, 0), v(3.2, 3.2, 0.7), "#2e333b", false),
				part(models.PartBox, v(0.9, 0, -3), v(0, 0, 0), v(1.2, 0.12, 0.2), "#2e333b", true),
				part(models.PartBox, v(0, 0.9, -3), v(0, 0, 0), v(0.12, 1.2, 0.2), "#2e333b", false),
				part(models.PartBox, v(0, -0.9, -3), v(0, 0, 0), v(0.12, 1.2, 0.2), "#2e333b", false),
				part(models.PartTube, v(0, 0, -2.6), v(math.Pi/2, 0, 0), v(0.9, 0.6, 0.9), "#2e333b", false),
				part(models.PartSphere, v(0, 0, -2.9), v(0, 0, 0), v(0.7, 0.7, 0.3), "#80deea", false),
			}},
		},
	},
	{
		Name: "Bulwark",
		Layers: []models.Layer3d{
			{Name: "Hull", Visible: true, Parts: []models.Part{
				part(models.PartBox, v(0, 0, 0), v(0, 0, 0), v(4, 1.6, 7), "#5f6b4f", false),
				part(models.PartWedge, v(0, 0, 4.3), v(0, math.Pi, 0), v(4, 1.6, 1.6), "#5f6b4f", false),
				part(models.PartPrism, v(0, 1.1, 0.5), v(math.Pi/2, 0, 0), v(2.2, 5, 0.8), "#7f8c6a", false),
				part(models.PartBox, v(0, 1.6, 2.6), v(0, 0, 0), v(1.4, 0.6, 1.2), "#7f8c6a", false),
				part(models.PartBox, v(0, 1.7, 3.15), v(0, 0, 0), v(1.1, 0.25, 0.1), "#ffcc80", false),
				part(models.PartBox, v(0, -0.95, 0), v(0, 0, 0), v(2.8, 0.3, 5.6), "#2f3529", false),
			}},
			{Name: "Armour", Visible: true, Parts: []models.Part{
				part(models.PartBox, v(2.25, -0.1, 0.5), v(0, 0, 0.15), v(0.5, 1.6, 6), "#7f8c6a", true),
				part(models.PartPyramid, v(2.4, 0.4, -2.4), v(0, 0, 0), v(0.6, 0.6, 0.6), "#2f3529", true),
				part(models.PartBox, v(2.6, -0.3, 2.5), v(0, 0, 0), v(0.3, 0.3, 1.2), "#2f3529", true),
				part(models.PartCylinder, v(2.6, -0.3, 3.4), v(math.Pi/2, 0, 0), v(0.15, 1, 0.15), "#3a3f44", true),
			}},
			{Name: "Turrets", Visible: true, Parts: []models.Part{
				part(models.PartDome, v(1.2, 0.8, 1.6), v(0, 0, 0), v(1, 0.6, 1), "#7f8c6a", true),
				part(models.PartBox, v(1.2, 1.15, 1.9), v(0, 0, 0), v(0.5, 0.3, 0.6), "#3a3f44", true),
				part(models.PartCylinder, v(1.05, 1.15, 2.7), v(math.Pi/2, 0, 0), v(0.12, 1.4, 0.12), "#3a3f44", true),
				part(models.PartCylinder, v(1.35, 1.15, 2.7), v(math.Pi/2, 0, 0), v(0.12, 1.4, 0.12), "#3a3f44", true),
				part(models.PartDome, v(1.2, 0.8, -1.2), v(0, 0, 0), v(1, 0.6, 1), "#7f8c6a", true),
				part(models.PartBox, v(1.2, 1.15, -0.9), v(0, 0, 0), v(0.5, 0.3, 0.6), "#3a3f44", true),
				part(models.PartCylinder, v(1.05, 1.15, -0.1), v(math.Pi/2, 0, 0), v(0.12, 1.4, 0.12), "#3a3f44", true),
				part(models.PartCylinder, v(1.35, 1.15, -0.1), v(math.Pi/2, 0, 0), v(0.12, 1.4, 0.12), "#3a3f44", true),
			}},
			{Name: "Engines", Visible: true, Parts: []models.Part{
				part(models.PartBox, v(0, 0, -4), v(0, 0, 0), v(3.6, 1.4, 1), "#2f3529", false),
				part(models.PartTube, v(1.1, 0, -4.7), v(math.Pi/2, 0, 0), v(0.9, 0.6, 0.9), "#3a3f44", true),
				part(models.PartSphere, v(1.1, 0, -4.9), v(0, 0, 0), v(0.65, 0.65, 0.2), "#ff7043", true),
				part(models.PartTube, v(0, 0.2, -4.7), v(math.Pi/2, 0, 0), v(0.7, 0.5, 0.7), "#3a3f44", false),
				part(models.PartSphere, v(0, 0.2, -4.85), v(0, 0, 0), v(0.5, 0.5, 0.2), "#ff7043", false),
			}},
		},
	},
	{
		Name: "Wisp",
		Layers: []models.Layer3d{
			{Name: "Body", Visible: true, Parts: []models.Part{
				part(models.PartSphere, v(0, 0, 0.4), v(0, 0, 0), v(2, 1.1, 3.4), "#eceff1", false),
				part(models.PartCone, v(0, 0, -2.3), v(-math.Pi/2, 0, 0), v(1.3, 2.2, 0.8), "#eceff1", false),
				part(models.PartDome, v(0, 0.45, 0.9), v(0, 0, 0), v(1, 0.6, 1.4), "#b39ddb", false),
				part(models.PartTorus, v(0, 0, 0.2), v(math.Pi/2, 0, 0), v(4.6, 4.6, 0.6), "#f48fb1", false),
				part(models.PartSphere, v(0, -0.55, 0.2), v(0, 0, 0), v(0.6, 0.3, 0.6), "#37474f", false),
			}},
			{Name: "Vanes", Visible: true, Parts: []models.Part{
				part(models.PartWedge, v(1.5, 0.1, -1), v(0, 0.4, -0.5), v(1.4, 0.08, 1.8), "#f48fb1", true),
				part(models.PartSphere, v(2.2, 0, 0.2), v(0, 0, 0), v(0.35, 0.35, 0.35), "#e1bee7", true),
				part(models.PartCapsule, v(1.1, -0.35, -1.6), v(math.Pi/2, 0, 0), v(0.35, 1.2, 0.35), "#37474f", true),
				part(models.PartSphere, v(1.1, -0.35, -2.3), v(0, 0, 0), v(0.28, 0.28, 0.12), "#e1bee7", true),
				part(models.PartCylinder, v(0, 0.9, -0.6), v(-0.4, 0, 0), v(0.05, 1.2, 0.05), "#37474f", false),
				part(models.PartSphere, v(0, 1.45, -0.85), v(0, 0, 0), v(0.12, 0.12, 0.12), "#e1bee7", false),
			}},
		},
	},
}

func v(x, y, z float64) models.Vec3 { return models.Vec3{x, y, z} }

func part(kind string, position, rotation, scale models.Vec3, color string, mirror bool) models.Part {
	return models.Part{Type: kind, Position: position, Rotation: rotation, Scale: scale, Color: color, Mirror: mirror}
}

// ShipInfo is a ship as the game sees it: an id, a name and the design.
type ShipInfo struct {
	Id     string           `json:"id"`
	Name   string           `json:"name"`
	Layers []models.Layer3d `json:"layers"`
}

func defaultShipId(project models.Project3d) string {
	return DefaultShipPrefix + strings.ToLower(project.Name)
}

// DefaultShips lists the built-in ships, offered to every player.
func DefaultShips() []ShipInfo {
	ships := make([]ShipInfo, 0, len(defaultShips))
	for _, project := range defaultShips {
		ships = append(ships, ShipInfo{Id: defaultShipId(project), Name: project.Name, Layers: project.Layers})
	}
	return ships
}

// DefaultShip returns the built-in ship with the given id.
func DefaultShip(id string) (*models.Project3d, bool) {
	for i := range defaultShips {
		if defaultShipId(defaultShips[i]) == id {
			project := defaultShips[i]
			return &project, true
		}
	}
	return nil, false
}
