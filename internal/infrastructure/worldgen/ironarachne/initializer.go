package ironarachne

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"

	ironconfig "github.com/ironarachne/world/config"
	ironrandom "github.com/ironarachne/world/pkg/random"
	irontown "github.com/ironarachne/world/pkg/town"
	ironworld "github.com/ironarachne/world/pkg/world"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
)

type Initializer struct {
	ids port.IDGenerator
}

func NewInitializer(ids port.IDGenerator) *Initializer {
	return &Initializer{ids: ids}
}

func (i *Initializer) Initialize(ctx context.Context, input port.WorldInitializationInput) (result port.WorldInitializationResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("ironarachne world generation panicked: %v", recovered)
		}
	}()
	if err := ctx.Err(); err != nil {
		return port.WorldInitializationResult{}, err
	}
	if err := ensureIronarachneConfig(); err != nil {
		return port.WorldInitializationResult{}, err
	}
	count := input.LocationCount
	if count <= 0 {
		count = 15
	}
	seed := strings.TrimSpace(input.Seed)
	if seed == "" {
		seed = input.ProjectID + ":" + input.SetupRun.RunID
	}
	if err := ironrandom.SeedFromString(seed); err != nil {
		return port.WorldInitializationResult{}, err
	}
	generated, err := ironworld.Generate()
	if err != nil {
		return port.WorldInitializationResult{}, err
	}
	height := len(generated.Tiles)
	width := 0
	if height > 0 {
		width = len(generated.Tiles[0])
	}
	if width == 0 || height == 0 {
		return port.WorldInitializationResult{}, fmt.Errorf("generated world has no tiles")
	}
	worldMap := model.WorldMap{
		ID:        i.newID("map"),
		ProjectID: input.ProjectID,
		Name:      firstNonEmpty(generated.Name, "Generated World"),
		Seed:      seed,
		Width:     width,
		Height:    height,
		Status:    "active",
		Properties: map[string]any{
			"source":            "github.com/ironarachne/world",
			"configured_width":  input.MapWidth,
			"configured_height": input.MapHeight,
		},
		CreatedAt: input.CurrentTime,
		UpdatedAt: input.CurrentTime,
	}
	points := sampleLandPoints(generated.Tiles, count)
	towns := generateTowns(count)
	locations := make([]model.LocationState, 0, count)
	factions := make([]model.FactionInfluence, 0, count)
	for idx, point := range points {
		tile := generated.Tiles[point.y][point.x]
		town := towns[idx]
		name := firstNonEmpty(town.Name, fmt.Sprintf("地点 %02d", idx+1))
		locationID := i.newID("location")
		locationType := firstNonEmpty(town.Category, "settlement")
		description := townDescription(town, tile)
		locations = append(locations, model.LocationState{
			ID:          locationID,
			ProjectID:   input.ProjectID,
			MapID:       worldMap.ID,
			RegionID:    fmt.Sprintf("region_%02d", idx+1),
			Name:        name,
			Type:        locationType,
			Description: description,
			X:           point.x,
			Y:           point.y,
			Radius:      8,
			Status:      "active",
			Properties: map[string]any{
				"source":           "github.com/ironarachne/world",
				"population":       town.Population,
				"building_style":   town.BuildingStyle,
				"climate":          town.Climate,
				"dominant_culture": town.DominantCulture,
				"exports":          town.Exports,
				"imports":          town.Imports,
				"tile": map[string]any{
					"altitude":    tile.Altitude,
					"temperature": tile.Temperature,
					"humidity":    tile.Humidity,
					"is_ocean":    tile.IsOcean,
					"terrain":     terrainName(tile),
				},
			},
			CreatedAt: input.CurrentTime,
			UpdatedAt: input.CurrentTime,
		})
		factions = append(factions, model.FactionInfluence{
			ID:          i.newID("faction"),
			ProjectID:   input.ProjectID,
			LocationID:  locationID,
			FactionName: firstNonEmpty(town.DominantCulture, name+"议会"),
			Influence:   6,
			Attitude:    "local_power",
			Description: firstNonEmpty(town.DominantCulture, name+"本地势力") + "影响此地的秩序、贸易和传闻流向。",
			Status:      "active",
			CreatedAt:   input.CurrentTime,
			UpdatedAt:   input.CurrentTime,
		})
	}
	tiles := make([]model.MapTile, 0, len(points))
	for _, point := range points {
		tile := generated.Tiles[point.y][point.x]
		tiles = append(tiles, model.MapTile{
			ID:          i.newID("tile"),
			ProjectID:   input.ProjectID,
			MapID:       worldMap.ID,
			X:           point.x,
			Y:           point.y,
			Altitude:    tile.Altitude,
			Temperature: tile.Temperature,
			Humidity:    tile.Humidity,
			IsOcean:     tile.IsOcean,
			Terrain:     terrainName(tile),
			Properties:  map[string]any{"sampled_location_tile": true},
			CreatedAt:   input.CurrentTime,
			UpdatedAt:   input.CurrentTime,
		})
	}
	states := make([]model.CharacterRuntimeState, 0, len(input.Characters))
	if len(locations) > 0 {
		for idx, character := range input.Characters {
			location := locations[idx%len(locations)]
			states = append(states, model.CharacterRuntimeState{
				CharacterID: character.ID,
				Tier:        "main",
				LocationKey: location.ID,
				X:           location.X,
				Y:           location.Y,
				Status:      "active",
			})
		}
	}
	stateByCharacter := make(map[string]model.CharacterRuntimeState, len(states))
	for _, state := range states {
		stateByCharacter[state.CharacterID] = state
	}
	worldState := make(map[string]model.WorldStateEntry, len(input.SetupDraft.WorldState))
	for _, entry := range input.SetupDraft.WorldState {
		if entry.Key == "" {
			continue
		}
		worldState[entry.Key] = entry
	}
	snapshot := model.WorldSnapshot{
		StoryTime:     input.CurrentTime,
		WorldState:    worldState,
		Characters:    stateByCharacter,
		Relationships: map[string]model.Relationship{},
		Factions:      factions,
		Locations:     locations,
	}
	return port.WorldInitializationResult{Map: worldMap, Tiles: tiles, Locations: locations, Factions: factions, CharacterStates: states, Snapshot: snapshot}, nil
}

func (i *Initializer) newID(prefix string) string {
	if i.ids != nil {
		return i.ids.New(prefix)
	}
	return fmt.Sprintf("%s_%d", prefix, rand.Int63())
}

var ironConfigMu sync.Mutex

func ensureIronarachneConfig() error {
	ironConfigMu.Lock()
	defer ironConfigMu.Unlock()
	if ironconfig.Cfg != nil && strings.TrimSpace(ironconfig.Cfg.WorldDataDirectory) != "" {
		return nil
	}
	moduleRoot, err := ironarachneModuleRoot()
	if err != nil {
		return err
	}
	ironconfig.Cfg = &ironconfig.Config{WorldDataDirectory: moduleRoot}
	return nil
}

func ironarachneModuleRoot() (string, error) {
	version := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/ironarachne/world" {
				version = dep.Version
				break
			}
		}
	}
	candidates := make([]string, 0, 3)
	if version != "" {
		if modCache := os.Getenv("GOMODCACHE"); modCache != "" {
			candidates = append(candidates, filepath.Join(modCache, "github.com", "ironarachne", "world@"+version))
		}
		if goPath := os.Getenv("GOPATH"); goPath != "" {
			candidates = append(candidates, filepath.Join(goPath, "pkg", "mod", "github.com", "ironarachne", "world@"+version))
		}
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, "go", "pkg", "mod", "github.com", "ironarachne", "world@"+version))
		}
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "data", "biomes.json")); err == nil {
			return candidate, nil
		}
	}
	if version == "" {
		return "", fmt.Errorf("github.com/ironarachne/world dependency version not found")
	}
	return "", fmt.Errorf("github.com/ironarachne/world data directory not found for %s", version)
}

type gridPoint struct {
	x int
	y int
}

func sampleLandPoints(tiles [][]ironworld.Tile, count int) []gridPoint {
	height := len(tiles)
	width := len(tiles[0])
	candidates := make([]gridPoint, 0, count)
	for y := height / 10; y < height && len(candidates) < count; y += maxInt(1, height/8) {
		for x := width / 10; x < width && len(candidates) < count; x += maxInt(1, width/8) {
			point := nearestLandPoint(tiles, x, y)
			if !containsPoint(candidates, point) {
				candidates = append(candidates, point)
			}
		}
	}
	for len(candidates) < count {
		point := nearestLandPoint(tiles, rand.Intn(width), rand.Intn(height))
		if !containsPoint(candidates, point) {
			candidates = append(candidates, point)
		}
	}
	return candidates[:count]
}

func nearestLandPoint(tiles [][]ironworld.Tile, x int, y int) gridPoint {
	height := len(tiles)
	width := len(tiles[0])
	if !tiles[y][x].IsOcean {
		return gridPoint{x: x, y: y}
	}
	maxRadius := maxInt(width, height)
	for radius := 1; radius < maxRadius; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if int(math.Abs(float64(dx))) != radius && int(math.Abs(float64(dy))) != radius {
					continue
				}
				nx := x + dx
				ny := y + dy
				if nx < 0 || ny < 0 || nx >= width || ny >= height {
					continue
				}
				if !tiles[ny][nx].IsOcean {
					return gridPoint{x: nx, y: ny}
				}
			}
		}
	}
	return gridPoint{x: x, y: y}
}

func containsPoint(points []gridPoint, target gridPoint) bool {
	for _, point := range points {
		if point == target {
			return true
		}
	}
	return false
}

func generateTowns(count int) []irontown.SimplifiedTown {
	towns := make([]irontown.SimplifiedTown, 0, count)
	for len(towns) < count {
		town, err := irontown.RandomSimplified()
		if err != nil {
			break
		}
		towns = append(towns, town)
	}
	for len(towns) < count {
		idx := len(towns) + 1
		towns = append(towns, irontown.SimplifiedTown{Name: fmt.Sprintf("据点 %02d", idx), Category: "settlement"})
	}
	return towns
}

func townDescription(town irontown.SimplifiedTown, tile ironworld.Tile) string {
	parts := []string{}
	if town.Category != "" {
		parts = append(parts, town.Category)
	}
	if town.Climate != "" {
		parts = append(parts, town.Climate)
	}
	if town.DominantCulture != "" {
		parts = append(parts, town.DominantCulture+"文化影响明显")
	}
	parts = append(parts, terrainName(tile))
	if town.Population > 0 {
		parts = append(parts, fmt.Sprintf("人口约 %d", town.Population))
	}
	return strings.Join(parts, "，")
}

func terrainName(tile ironworld.Tile) string {
	if tile.IsOcean {
		return "ocean"
	}
	if tile.Altitude > 70 {
		return "mountain"
	}
	if tile.Altitude > 35 {
		return "highland"
	}
	if tile.Humidity > 80 {
		return "wetland"
	}
	if tile.Temperature > 75 && tile.Humidity < 35 {
		return "dryland"
	}
	return "plain"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

var _ port.WorldInitializer = (*Initializer)(nil)
