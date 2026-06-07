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
	"time"

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
	areas := i.generateMapAreas(input.ProjectID, worldMap.ID, generated.Tiles, input.CurrentTime)
	regionLocations := i.regionLocations(input.ProjectID, worldMap.ID, areas, input.CurrentTime)
	points := sampleLandPoints(generated.Tiles, count)
	towns := generateTowns(count)
	locations := make([]model.LocationState, 0, len(regionLocations)+count)
	locations = append(locations, regionLocations...)
	factions := make([]model.FactionInfluence, 0, count)
	for idx, point := range points {
		tile := generated.Tiles[point.y][point.x]
		town := towns[idx]
		name := firstNonEmpty(town.Name, fmt.Sprintf("地点 %02d", idx+1))
		sector := areaForPoint(areas, "sector", point.x, point.y)
		region := parentAreaFor(areas, sector)
		regionLocationID := regionLocationIDForArea(regionLocations, region.ID)
		locationID := i.newID("location")
		locationType := firstNonEmpty(town.Category, "settlement")
		description := townDescription(town, tile)
		locations = append(locations, model.LocationState{
			ID:               locationID,
			ProjectID:        input.ProjectID,
			MapID:            worldMap.ID,
			AreaID:           sector.ID,
			RegionID:         regionLocationID,
			ParentLocationID: regionLocationID,
			Name:             name,
			Type:             locationType,
			Scale:            model.LocationScaleSettlement,
			DetailState:      model.LocationDetailStub,
			Description:      description,
			X:                point.x,
			Y:                point.y,
			Radius:           8,
			Status:           "active",
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
	settlementLocations := locations[len(regionLocations):]
	if len(settlementLocations) > 0 {
		for idx, character := range input.Characters {
			location := settlementLocations[idx%len(settlementLocations)]
			states = append(states, model.CharacterRuntimeState{
				CharacterID: character.ID,
				Tier:        inferInitialCharacterTier(character),
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
		Areas:         areas,
		Factions:      factions,
		Locations:     locations,
	}
	return port.WorldInitializationResult{Map: worldMap, Areas: areas, Tiles: tiles, Locations: locations, Factions: factions, CharacterStates: states, Snapshot: snapshot}, nil
}

func (i *Initializer) newID(prefix string) string {
	if i.ids != nil {
		return i.ids.New(prefix)
	}
	return fmt.Sprintf("%s_%d", prefix, rand.Int63())
}

func (i *Initializer) generateMapAreas(projectID string, mapID string, tiles [][]ironworld.Tile, now time.Time) []model.MapArea {
	height := len(tiles)
	width := len(tiles[0])
	areas := make([]model.MapArea, 0, 20)
	regions := gridAreas(projectID, mapID, "", "region", width, height, 2, 2, now, i.newID)
	for idx := range regions {
		regions[idx].DominantTerrain = terrainName(tiles[clampInt(regions[idx].CenterY, 0, height-1)][clampInt(regions[idx].CenterX, 0, width-1)])
	}
	areas = append(areas, regions...)
	for _, region := range regions {
		sectors := gridAreas(projectID, mapID, region.ID, "sector", region.MaxX-region.MinX+1, region.MaxY-region.MinY+1, 2, 2, now, i.newID)
		for idx := range sectors {
			sectors[idx].MinX += region.MinX
			sectors[idx].MaxX += region.MinX
			sectors[idx].CenterX += region.MinX
			sectors[idx].MinY += region.MinY
			sectors[idx].MaxY += region.MinY
			sectors[idx].CenterY += region.MinY
			sectors[idx].DominantTerrain = terrainName(tiles[clampInt(sectors[idx].CenterY, 0, height-1)][clampInt(sectors[idx].CenterX, 0, width-1)])
		}
		areas = append(areas, sectors...)
	}
	return areas
}

func gridAreas(projectID, mapID, parentID, level string, width, height, cols, rows int, now time.Time, newID func(string) string) []model.MapArea {
	areas := make([]model.MapArea, 0, cols*rows)
	cellW := maxInt(1, width/cols)
	cellH := maxInt(1, height/rows)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			minX := col * cellW
			minY := row * cellH
			maxX := minX + cellW - 1
			maxY := minY + cellH - 1
			if col == cols-1 {
				maxX = width - 1
			}
			if row == rows-1 {
				maxY = height - 1
			}
			areas = append(areas, model.MapArea{
				ID:           newID("area"),
				ProjectID:    projectID,
				MapID:        mapID,
				ParentAreaID: parentID,
				Name:         fmt.Sprintf("%s %d-%d", level, row+1, col+1),
				Level:        level,
				MinX:         minX,
				MinY:         minY,
				MaxX:         maxX,
				MaxY:         maxY,
				CenterX:      (minX + maxX) / 2,
				CenterY:      (minY + maxY) / 2,
				Status:       "active",
				Properties:   map[string]any{"source": "grid_partition"},
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}
	}
	return areas
}

func (i *Initializer) regionLocations(projectID string, mapID string, areas []model.MapArea, now time.Time) []model.LocationState {
	locations := []model.LocationState{}
	for _, area := range areas {
		if area.Level != "region" {
			continue
		}
		locations = append(locations, model.LocationState{
			ID:          i.newID("location"),
			ProjectID:   projectID,
			MapID:       mapID,
			AreaID:      area.ID,
			RegionID:    area.ID,
			Name:        area.Name,
			Type:        "region",
			Scale:       model.LocationScaleRegion,
			DetailState: model.LocationDetailStub,
			Description: fmt.Sprintf("%s is a broad world region.", area.Name),
			X:           area.CenterX,
			Y:           area.CenterY,
			Radius:      maxInt(8, (area.MaxX-area.MinX+area.MaxY-area.MinY)/4),
			Status:      "active",
			Properties:  map[string]any{"area_id": area.ID, "public_summary": "A broad region that contains settlements and local routes."},
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return locations
}

func areaForPoint(areas []model.MapArea, level string, x int, y int) model.MapArea {
	for _, area := range areas {
		if area.Level == level && x >= area.MinX && x <= area.MaxX && y >= area.MinY && y <= area.MaxY {
			return area
		}
	}
	return model.MapArea{}
}

func parentAreaFor(areas []model.MapArea, child model.MapArea) model.MapArea {
	if child.ParentAreaID == "" {
		return child
	}
	for _, area := range areas {
		if area.ID == child.ParentAreaID {
			return area
		}
	}
	return model.MapArea{}
}

func regionLocationIDForArea(locations []model.LocationState, areaID string) string {
	for _, location := range locations {
		if location.AreaID == areaID {
			return location.ID
		}
	}
	return ""
}

func inferInitialCharacterTier(character model.Character) string {
	role := strings.ToLower(strings.TrimSpace(character.Role))
	switch {
	case strings.Contains(role, "protagonist"), strings.Contains(role, "主角"), strings.Contains(role, "lead"), strings.Contains(role, "核心反派"), strings.Contains(role, "最终"), strings.Contains(role, "boss"):
		return "tier_1"
	case strings.Contains(role, "antagonist"), strings.Contains(role, "反派"), strings.Contains(role, "villain"):
		return "tier_1"
	case strings.Contains(role, "minor"), strings.Contains(role, "background"), strings.Contains(role, "路人"), strings.Contains(role, "背景"):
		return "tier_3"
	default:
		return "tier_2"
	}
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

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

var _ port.WorldInitializer = (*Initializer)(nil)
