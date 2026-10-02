package domain

import (
	"encoding/json"
	"fmt"
)

const (
	MapWidth  = 500
	MapHeight = 500
	TileSize  = 16
	MapOrigin = -4000
)

// CollisionMap stores blocked map cells.
type CollisionMap struct {
	blocked [MapHeight][MapWidth]bool
}

type tiledMap struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	Layers []struct {
		Name string `json:"name"`
		Data []int  `json:"data"`
	} `json:"layers"`
}

// LoadCollisionMap parses the Tiled map's "predios" layer.
func LoadCollisionMap(data []byte) (CollisionMap, error) {
	var source tiledMap
	if err := json.Unmarshal(data, &source); err != nil {
		return CollisionMap{}, fmt.Errorf("parse map: %w", err)
	}
	if source.Width != MapWidth || source.Height != MapHeight {
		return CollisionMap{}, fmt.Errorf("map dimensions are %dx%d, want %dx%d", source.Width, source.Height, MapWidth, MapHeight)
	}

	var result CollisionMap
	for _, layer := range source.Layers {
		if layer.Name != "predios" {
			continue
		}
		if len(layer.Data) != MapWidth*MapHeight {
			return CollisionMap{}, fmt.Errorf("predios layer has %d cells, want %d", len(layer.Data), MapWidth*MapHeight)
		}
		for index, tile := range layer.Data {
			result.blocked[index/MapWidth][index%MapWidth] = tile != 0
		}
		return result, nil
	}
	return CollisionMap{}, fmt.Errorf(`map layer "predios" not found`)
}

func (m CollisionMap) blockedAt(x, y float64) bool {
	col, row := worldToTile(x, y)
	if col < 0 || col >= MapWidth || row < 0 || row >= MapHeight {
		return true
	}
	return m.blocked[row][col]
}

func worldToTile(x, y float64) (int, int) {
	return intFloor((x - MapOrigin) / TileSize), intFloor((y - MapOrigin) / TileSize)
}

func intFloor(value float64) int {
	if value < 0 && value != float64(int(value)) {
		return int(value) - 1
	}
	return int(value)
}
