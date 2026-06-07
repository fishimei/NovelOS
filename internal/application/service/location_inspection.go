package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

const defaultReachableLocationLimit = 8

type LocationInspectionService struct {
	events       port.StoryEventStore
	clock        port.Clock
	ids          port.IDGenerator
	subdivider   port.LocationSubdivisionGenerator
	nearbyRadius int
}

type LocationInspectionOption func(*LocationInspectionService)

func WithLocationSubdivisionGenerator(generator port.LocationSubdivisionGenerator) LocationInspectionOption {
	return func(s *LocationInspectionService) {
		s.subdivider = generator
	}
}

func WithLocationNearbyRadius(radius int) LocationInspectionOption {
	return func(s *LocationInspectionService) {
		if radius > 0 {
			s.nearbyRadius = radius
		}
	}
}

func NewLocationInspectionService(events port.StoryEventStore, clock port.Clock, ids port.IDGenerator, options ...LocationInspectionOption) *LocationInspectionService {
	service := &LocationInspectionService{events: events, clock: clock, ids: ids, nearbyRadius: 25}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *LocationInspectionService) EnsureReachableLocations(ctx context.Context, input model.LocationReachabilityInput) (model.LocationInspectionContext, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	currentID := strings.TrimSpace(input.CurrentLocationID)
	if currentID == "" && input.CharacterID != "" {
		if state, ok := input.World.Characters[input.CharacterID]; ok {
			currentID = strings.TrimSpace(state.LocationKey)
		}
	}
	if currentID == "" {
		return model.LocationInspectionContext{}, pkgerr.Validation("current_location_id is required")
	}
	locations, err := s.projectLocations(ctx, projectID, input.World)
	if err != nil {
		return model.LocationInspectionContext{}, err
	}
	current, ok := locationByID(locations, currentID)
	if !ok && projectID != "" && s.events != nil {
		current, err = s.events.GetLocation(ctx, projectID, currentID)
		if err != nil {
			return model.LocationInspectionContext{}, err
		}
		ok = true
	}
	if !ok {
		return model.LocationInspectionContext{}, pkgerr.NotFound("current location not found")
	}
	if projectID == "" {
		projectID = current.ProjectID
	}
	if projectID == "" {
		return model.LocationInspectionContext{}, pkgerr.Validation("project_id is required")
	}
	influences, _ := s.factionInfluences(ctx, projectID)
	locations, current, err = s.ensureCurrentLocationExpanded(ctx, projectID, locations, current, input.World, influences)
	if err != nil {
		return model.LocationInspectionContext{}, err
	}
	areas, _ := s.projectAreas(ctx, projectID, input.World)
	return model.LocationInspectionContext{
		CurrentLocation:    visibleLocation(current),
		ReachableLocations: reachableLocationContexts(locations, areas, influences, current, defaultReachableLocationLimit, s.nearbyRadius),
	}, nil
}

func (s *LocationInspectionService) InspectLocation(ctx context.Context, input model.LocationInspectionInput) (model.LocationInspectionResult, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	locationID := strings.TrimSpace(input.LocationID)
	if projectID == "" {
		return model.LocationInspectionResult{}, pkgerr.Validation("project_id is required")
	}
	if locationID == "" {
		return model.LocationInspectionResult{}, pkgerr.Validation("location_id is required")
	}
	contextInput := model.LocationReachabilityInput{
		ProjectID:         projectID,
		CharacterID:       input.CharacterID,
		CurrentLocationID: input.CurrentLocationID,
		World:             input.World,
	}
	reachability, err := s.EnsureReachableLocations(ctx, contextInput)
	if err != nil {
		return model.LocationInspectionResult{}, err
	}
	if !locationIsReachable(reachability, locationID) {
		return model.LocationInspectionResult{}, pkgerr.Validation("location is not reachable from current location")
	}
	location, err := s.getLocation(ctx, projectID, locationID, input.World)
	if err != nil {
		return model.LocationInspectionResult{}, err
	}
	locations, err := s.projectLocations(ctx, projectID, input.World)
	if err != nil {
		return model.LocationInspectionResult{}, err
	}
	influences, _ := s.factionInfluences(ctx, projectID)
	areas, _ := s.projectAreas(ctx, projectID, input.World)
	changed, generated, err := s.materializeLocation(ctx, projectID, locations, input.World, influences, location, strings.TrimSpace(input.Reason))
	if err != nil {
		return model.LocationInspectionResult{}, err
	}
	if len(changed) > 0 {
		if err := s.upsertLocations(ctx, projectID, changed); err != nil {
			return model.LocationInspectionResult{}, err
		}
		locations, err = s.projectLocations(ctx, projectID, input.World)
		if err != nil {
			return model.LocationInspectionResult{}, err
		}
		if updated, ok := locationByID(locations, location.ID); ok {
			location = updated
		}
	}
	childs := childLocations(locations, location.ID)
	return model.LocationInspectionResult{
		CurrentLocation:    visibleLocation(reachability.CurrentLocation),
		InspectedLocation:  visibleLocation(location),
		Ancestors:          visibleLocations(ancestorLocations(locations, location)),
		ChildLocations:     visibleLocations(childs),
		ReachableLocations: reachableLocationContexts(locations, areas, influences, location, defaultReachableLocationLimit, s.nearbyRadius),
		Generated:          generated,
	}, nil
}

func (s *LocationInspectionService) projectAreas(ctx context.Context, projectID string, world model.WorldSnapshot) ([]model.MapArea, error) {
	if s.events != nil && strings.TrimSpace(projectID) != "" {
		areas, err := s.events.ListMapAreasByProjectID(ctx, projectID)
		if err != nil {
			return nil, err
		}
		return areas, nil
	}
	return append([]model.MapArea(nil), world.Areas...), nil
}

func (s *LocationInspectionService) projectLocations(ctx context.Context, projectID string, world model.WorldSnapshot) ([]model.LocationState, error) {
	if s.events != nil && strings.TrimSpace(projectID) != "" {
		locations, err := s.events.ListLocationsByProjectID(ctx, projectID)
		if err != nil {
			return nil, err
		}
		return locations, nil
	}
	return append([]model.LocationState(nil), world.Locations...), nil
}

func (s *LocationInspectionService) getLocation(ctx context.Context, projectID string, locationID string, world model.WorldSnapshot) (model.LocationState, error) {
	if s.events != nil {
		return s.events.GetLocation(ctx, projectID, locationID)
	}
	if location, ok := locationByID(world.Locations, locationID); ok {
		return location, nil
	}
	return model.LocationState{}, pkgerr.NotFound("location state not found")
}

func (s *LocationInspectionService) factionInfluences(ctx context.Context, projectID string) ([]model.FactionInfluence, error) {
	if s.events == nil || strings.TrimSpace(projectID) == "" {
		return nil, nil
	}
	return s.events.ListFactionInfluencesByProjectID(ctx, projectID)
}

func (s *LocationInspectionService) ensureCurrentLocationExpanded(ctx context.Context, projectID string, locations []model.LocationState, current model.LocationState, world model.WorldSnapshot, influences []model.FactionInfluence) ([]model.LocationState, model.LocationState, error) {
	current = normalizeLocation(current)
	changed, _, err := s.materializeLocation(ctx, projectID, locations, world, influences, current, "current location context")
	if err != nil {
		return nil, model.LocationState{}, err
	}
	if len(changed) > 0 {
		if err := s.upsertLocations(ctx, projectID, changed); err != nil {
			return nil, model.LocationState{}, err
		}
		if s.events != nil {
			refreshed, err := s.events.ListLocationsByProjectID(ctx, projectID)
			if err != nil {
				return nil, model.LocationState{}, err
			}
			locations = refreshed
			if updated, ok := locationByID(locations, current.ID); ok {
				current = updated
			}
		} else {
			locations = mergeLocations(locations, changed)
		}
	}
	return locations, current, nil
}

func (s *LocationInspectionService) upsertLocations(ctx context.Context, projectID string, locations []model.LocationState) error {
	if s.events == nil || len(locations) == 0 {
		return nil
	}
	return s.events.UpsertLocations(ctx, projectID, locations)
}

func (s *LocationInspectionService) materializeLocation(ctx context.Context, projectID string, locations []model.LocationState, world model.WorldSnapshot, influences []model.FactionInfluence, location model.LocationState, reason string) ([]model.LocationState, bool, error) {
	location = normalizeLocation(location)
	children := childLocations(locations, location.ID)
	needsDetail := normalizedDetailState(location) != model.LocationDetailInitialized
	needsChildren := canHaveChildLocations(location) && !locationChildrenExpanded(location) && len(children) == 0
	needsChildMarker := canHaveChildLocations(location) && !locationChildrenExpanded(location) && len(children) > 0
	if !needsDetail && !needsChildren && !needsChildMarker {
		return nil, false, nil
	}
	changed := []model.LocationState{}
	generated := false
	if needsChildMarker && !needsDetail && !needsChildren {
		location.Properties = cloneMap(location.Properties)
		location.Properties["children_expanded"] = true
		location.UpdatedAt = currentTime(s.clock)
		return []model.LocationState{location}, false, nil
	}
	if needsDetail || needsChildren {
		if s.subdivider == nil {
			location = s.initializeLocation(location, reason)
			if needsChildren {
				newChildren := s.generateChildLocationStubs(projectID, location)
				if len(newChildren) == 0 {
					return nil, false, pkgerr.Internal("location subdivision generator is required", nil)
				}
				location.Properties = cloneMap(location.Properties)
				location.Properties["children_expanded"] = true
				changed = append(changed, location)
				changed = append(changed, newChildren...)
				return changed, false, nil
			}
			return []model.LocationState{location}, false, nil
		}
		areas, _ := s.projectAreas(ctx, projectID, world)
		plan, err := s.subdivider.GenerateLocationSubdivision(ctx, model.LocationSubdivisionInput{
			ProjectID:         projectID,
			ParentLocation:    location,
			Area:              areaByID(areas, location.AreaID),
			ExistingChildren:  children,
			SiblingLocations:  siblingLocations(locations, location),
			World:             world,
			FactionInfluences: factionInfluencesForLocation(influences, location.ID),
			Reason:            reason,
			NeedChildren:      needsChildren,
		})
		if err != nil {
			return nil, false, err
		}
		location = applyLocationDetailPatch(location, plan.Detail)
		if needsDetail {
			location = s.initializeLocation(location, reason)
		} else {
			location.UpdatedAt = currentTime(s.clock)
		}
		if needsChildren {
			newChildren := s.locationsFromSubdivisionChildren(projectID, location, children, plan.Children)
			if len(newChildren) == 0 {
				return nil, false, pkgerr.Validation("location subdivision produced no valid child locations")
			}
			location.Properties = cloneMap(location.Properties)
			location.Properties["children_expanded"] = true
			changed = append(changed, location)
			changed = append(changed, newChildren...)
		} else {
			changed = append(changed, location)
		}
		generated = true
	}
	return changed, generated, nil
}

func applyLocationDetailPatch(location model.LocationState, patch model.LocationDetailPatch) model.LocationState {
	if strings.TrimSpace(patch.Name) != "" {
		location.Name = strings.TrimSpace(patch.Name)
	}
	if strings.TrimSpace(patch.Type) != "" {
		location.Type = strings.TrimSpace(patch.Type)
	}
	if strings.TrimSpace(patch.Description) != "" {
		location.Description = strings.TrimSpace(patch.Description)
	}
	location.Properties = cloneMap(location.Properties)
	for key, value := range patch.Properties {
		location.Properties[key] = value
	}
	return location
}

func (s *LocationInspectionService) locationsFromSubdivisionChildren(projectID string, parent model.LocationState, existing []model.LocationState, children []model.LocationSubdivisionChild) []model.LocationState {
	if !canHaveChildLocations(parent) {
		return nil
	}
	seenNames := map[string]struct{}{}
	for _, child := range existing {
		seenNames[strings.ToLower(strings.TrimSpace(child.Name))] = struct{}{}
	}
	out := []model.LocationState{}
	for idx, child := range children {
		name := strings.TrimSpace(child.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seenNames[key]; ok {
			continue
		}
		seenNames[key] = struct{}{}
		scale := normalizeChildScale(parent, child.Scale)
		if scale == "" {
			continue
		}
		now := currentTime(s.clock)
		radius := child.Radius
		if radius <= 0 {
			radius = maxInt(1, parent.Radius/2)
		}
		out = append(out, model.LocationState{
			ID:               generatedID(s.ids, s.clock, "location"),
			ProjectID:        projectID,
			MapID:            parent.MapID,
			AreaID:           parent.AreaID,
			RegionID:         firstNonBlank(parent.RegionID, parent.ID),
			ParentLocationID: parent.ID,
			Name:             name,
			Type:             firstNonBlank(child.Type, scale),
			Scale:            scale,
			DetailState:      model.LocationDetailStub,
			Description:      firstNonBlank(child.Description, defaultLocationDescription(model.LocationState{Name: name, Type: child.Type, Scale: scale})),
			X:                parent.X + child.DX,
			Y:                parent.Y + child.DY,
			Radius:           radius,
			Status:           "active",
			Properties: mergeStringAnyMaps(map[string]any{
				"generation_seed": fmt.Sprintf("%s:%d:%s", parent.ID, idx, key),
				"public_summary":  firstNonBlank(child.Description, name),
			}, child.Properties),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	return out
}

func (s *LocationInspectionService) expandLocation(location model.LocationState) model.LocationState {
	location = normalizeLocation(location)
	location.DetailState = model.LocationDetailExpanded
	location.Description = firstNonBlank(location.Description, defaultLocationDescription(location))
	location.Properties = cloneMap(location.Properties)
	location.Properties["public_summary"] = firstNonBlank(asString(location.Properties["public_summary"]), location.Description)
	location.Properties["affordances"] = firstStringSlice(location.Properties["affordances"], defaultAffordances(location))
	location.Properties["risks"] = firstStringSlice(location.Properties["risks"], defaultRisks(location))
	location.Properties["resources"] = firstStringSlice(location.Properties["resources"], defaultResources(location))
	location.Properties["generation_seed"] = firstNonBlank(asString(location.Properties["generation_seed"]), location.ID)
	location.UpdatedAt = currentTime(s.clock)
	if location.CreatedAt.IsZero() {
		location.CreatedAt = location.UpdatedAt
	}
	return location
}

func (s *LocationInspectionService) initializeLocation(location model.LocationState, reason string) model.LocationState {
	location = s.expandLocation(location)
	location.DetailState = model.LocationDetailInitialized
	location.Properties = cloneMap(location.Properties)
	location.Properties["access_rules"] = firstStringSlice(location.Properties["access_rules"], defaultAccessRules(location))
	location.Properties["initialized_reason"] = reason
	location.UpdatedAt = currentTime(s.clock)
	return location
}

func (s *LocationInspectionService) generateChildLocationStubs(projectID string, parent model.LocationState) []model.LocationState {
	specs := childSpecsFor(parent)
	children := make([]model.LocationState, 0, len(specs))
	for idx, spec := range specs {
		child := model.LocationState{
			ID:               generatedID(s.ids, s.clock, "location"),
			ProjectID:        projectID,
			MapID:            parent.MapID,
			RegionID:         parent.RegionID,
			ParentLocationID: parent.ID,
			Name:             strings.TrimSpace(parent.Name + " " + spec.name),
			Type:             spec.locationType,
			Scale:            spec.scale,
			DetailState:      model.LocationDetailStub,
			Description:      spec.summary,
			X:                parent.X + spec.dx,
			Y:                parent.Y + spec.dy,
			Radius:           maxInt(1, parent.Radius/2),
			Status:           "active",
			Properties: map[string]any{
				"public_summary":  spec.summary,
				"route_hint":      spec.route,
				"generation_seed": fmt.Sprintf("%s:%d:%s", parent.ID, idx, spec.locationType),
			},
			CreatedAt: currentTime(s.clock),
			UpdatedAt: currentTime(s.clock),
		}
		children = append(children, child)
	}
	return children
}

type childLocationSpec struct {
	name         string
	locationType string
	scale        string
	summary      string
	route        string
	dx           int
	dy           int
}

func childSpecsFor(parent model.LocationState) []childLocationSpec {
	switch normalizedScale(parent) {
	case model.LocationScaleRegion:
		return []childLocationSpec{
			{name: "Settlement", locationType: "settlement", scale: model.LocationScaleSettlement, summary: "A reachable settlement in this region.", route: "regional road", dx: -4, dy: 2},
			{name: "Crossing", locationType: "crossing", scale: model.LocationScaleSite, summary: "A route junction where travelers and news pass.", route: "regional road", dx: 3, dy: -2},
			{name: "Outpost", locationType: "outpost", scale: model.LocationScaleSite, summary: "A guarded local position with limited access.", route: "patrol path", dx: 5, dy: 1},
			{name: "Ruin", locationType: "ruin", scale: model.LocationScaleSite, summary: "An old place with uncertain value and risk.", route: "old trail", dx: -6, dy: -3},
			{name: "River Landing", locationType: "landing", scale: model.LocationScaleSite, summary: "A modest arrival point on a local route.", route: "water path", dx: 1, dy: 5},
			{name: "Watch Hill", locationType: "lookout", scale: model.LocationScaleSite, summary: "A high point used to observe movement.", route: "hill track", dx: -2, dy: -5},
		}
	case model.LocationScaleSettlement:
		return []childLocationSpec{
			{name: "Gate", locationType: "gate", scale: model.LocationScaleSite, summary: "The main controlled entry point.", route: "main street", dx: -3, dy: 0},
			{name: "Market District", locationType: "market", scale: model.LocationScaleDistrict, summary: "A public trade area where news and pressure spread.", route: "main street", dx: 2, dy: 1},
			{name: "Council Hall", locationType: "authority", scale: model.LocationScaleSite, summary: "The local authority seat and record center.", route: "civic road", dx: 1, dy: -2},
			{name: "Inn", locationType: "inn", scale: model.LocationScaleSite, summary: "A public resting place for travelers and meetings.", route: "side street", dx: -1, dy: 2},
			{name: "Shrine", locationType: "shrine", scale: model.LocationScaleSite, summary: "A quiet ritual site with local memory.", route: "stone path", dx: 4, dy: -1},
			{name: "Back Alley", locationType: "alley", scale: model.LocationScaleSite, summary: "A low-visibility passage used for private movement.", route: "narrow lane", dx: -4, dy: 2},
		}
	case model.LocationScaleDistrict:
		return []childLocationSpec{
			{name: "Main Street", locationType: "street", scale: model.LocationScaleSite, summary: "The visible center of movement.", route: "district path", dx: 0, dy: 1},
			{name: "Storehouse", locationType: "storehouse", scale: model.LocationScaleSite, summary: "A controlled place for goods and records.", route: "service lane", dx: -2, dy: 0},
			{name: "Tea House", locationType: "meeting_place", scale: model.LocationScaleSite, summary: "A semi-public place for talk and rumor.", route: "market lane", dx: 1, dy: 1},
			{name: "Residence Court", locationType: "residence", scale: model.LocationScaleSite, summary: "A private living cluster.", route: "inner lane", dx: 2, dy: -1},
			{name: "Workshop Row", locationType: "workshop", scale: model.LocationScaleSite, summary: "A practical row where skilled work leaves useful traces.", route: "work lane", dx: -1, dy: -2},
			{name: "Notice Board", locationType: "notice_board", scale: model.LocationScaleSite, summary: "A public information point with local warnings and offers.", route: "central lane", dx: 3, dy: 0},
		}
	case model.LocationScaleSite:
		return []childLocationSpec{
			{name: "Entry", locationType: "entry", scale: model.LocationScaleRoom, summary: "The first visible part of the site.", route: "inside", dx: 0, dy: 0},
			{name: "Inner Hall", locationType: "hall", scale: model.LocationScaleRoom, summary: "A controlled interior area.", route: "inside", dx: 1, dy: 0},
			{name: "Yard", locationType: "yard", scale: model.LocationScaleRoom, summary: "A small open interior space.", route: "inside", dx: 0, dy: 1},
			{name: "Back Room", locationType: "back_room", scale: model.LocationScaleRoom, summary: "A less visible interior room.", route: "inside", dx: -1, dy: 0},
			{name: "Storage", locationType: "storage", scale: model.LocationScaleRoom, summary: "A place where tools, goods, or records may be kept.", route: "inside", dx: 0, dy: -1},
			{name: "Side Exit", locationType: "side_exit", scale: model.LocationScaleRoom, summary: "An alternate passage with limited visibility.", route: "inside", dx: 1, dy: 1},
		}
	default:
		return nil
	}
}

func reachableLocationContexts(locations []model.LocationState, areas []model.MapArea, influences []model.FactionInfluence, current model.LocationState, limit int, nearbyRadius int) []model.NearbyLocationContext {
	candidates := []model.NearbyLocationContext{{
		Location: visibleLocation(current),
		Distance: 0,
		Route:    "stay",
		Relation: "current",
	}}
	for _, location := range locations {
		if location.ID == "" || location.ID == current.ID {
			continue
		}
		if locationAccessBlocked(location) {
			continue
		}
		relation := locationRelation(current, location)
		if relation == "" {
			relation = spatialRelation(areas, current, location, nearbyRadius)
		}
		if relation == "" {
			continue
		}
		candidates = append(candidates, model.NearbyLocationContext{
			Location:          visibleLocation(location),
			Distance:          locationDistanceValue(current, location),
			Route:             routeHint(location, relation),
			Relation:          relation,
			FactionInfluences: factionInfluencesForLocation(influences, location.ID),
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Relation == candidates[j].Relation {
			if candidates[i].Distance == candidates[j].Distance {
				return candidates[i].Location.ID < candidates[j].Location.ID
			}
			return candidates[i].Distance < candidates[j].Distance
		}
		return relationRank(candidates[i].Relation) < relationRank(candidates[j].Relation)
	})
	if limit <= 0 || len(candidates) <= limit {
		return candidates
	}
	return candidates[:limit]
}

func locationRelation(current model.LocationState, location model.LocationState) string {
	switch {
	case location.ParentLocationID == current.ID:
		return "child"
	case current.ParentLocationID != "" && location.ID == current.ParentLocationID:
		return "parent"
	case current.ParentLocationID != "" && location.ParentLocationID == current.ParentLocationID:
		return "sibling"
	case current.ParentLocationID == "" && location.ParentLocationID == "":
		return "nearby"
	default:
		return ""
	}
}

func relationRank(relation string) int {
	switch relation {
	case "current":
		return 0
	case "child":
		return 1
	case "parent":
		return 2
	case "sibling":
		return 3
	case "same_area":
		return 4
	case "adjacent_area":
		return 5
	case "nearby":
		return 6
	default:
		return 9
	}
}

func routeHint(location model.LocationState, relation string) string {
	if location.Properties != nil {
		if route := asString(location.Properties["route_hint"]); route != "" {
			return route
		}
	}
	return relation
}

func locationIsReachable(ctx model.LocationInspectionContext, locationID string) bool {
	if ctx.CurrentLocation.ID == locationID {
		return true
	}
	for _, reachable := range ctx.ReachableLocations {
		if reachable.Location.ID == locationID {
			return true
		}
	}
	return false
}

func spatialRelation(areas []model.MapArea, current model.LocationState, location model.LocationState, nearbyRadius int) string {
	if current.AreaID != "" && current.AreaID == location.AreaID {
		return "same_area"
	}
	if mapAreasAdjacent(areas, current.AreaID, location.AreaID) {
		return "adjacent_area"
	}
	if nearbyRadius > 0 && locationDistanceValue(current, location) <= float64(nearbyRadius) {
		return "nearby"
	}
	return ""
}

func mapAreasAdjacent(areas []model.MapArea, leftID string, rightID string) bool {
	if leftID == "" || rightID == "" || leftID == rightID {
		return false
	}
	left := areaByID(areas, leftID)
	right := areaByID(areas, rightID)
	if left.ID == "" || right.ID == "" {
		return false
	}
	return left.MinX <= right.MaxX+1 && left.MaxX+1 >= right.MinX && left.MinY <= right.MaxY+1 && left.MaxY+1 >= right.MinY
}

func locationAccessBlocked(location model.LocationState) bool {
	status := strings.ToLower(strings.TrimSpace(location.Status))
	if status == "blocked" || status == "closed" || status == "inactive" {
		return true
	}
	if location.Properties == nil {
		return false
	}
	access := strings.ToLower(asString(location.Properties["access"]))
	return access == "blocked" || access == "forbidden"
}

func normalizeLocation(location model.LocationState) model.LocationState {
	if location.Scale == "" {
		if location.ParentLocationID == "" {
			location.Scale = model.LocationScaleSettlement
		} else {
			location.Scale = model.LocationScaleSite
		}
	}
	if location.DetailState == "" {
		location.DetailState = model.LocationDetailStub
	}
	if location.Status == "" {
		location.Status = "active"
	}
	return location
}

func normalizedScale(location model.LocationState) string {
	return normalizeLocation(location).Scale
}

func normalizedDetailState(location model.LocationState) string {
	return normalizeLocation(location).DetailState
}

func canHaveChildLocations(location model.LocationState) bool {
	switch normalizedScale(location) {
	case model.LocationScaleRegion, model.LocationScaleSettlement, model.LocationScaleDistrict, model.LocationScaleSite:
		return true
	default:
		return false
	}
}

func locationChildrenExpanded(location model.LocationState) bool {
	if location.Properties == nil {
		return false
	}
	value, _ := location.Properties["children_expanded"].(bool)
	return value
}

func visibleLocation(location model.LocationState) model.LocationState {
	location.Properties = cloneMap(location.Properties)
	return location
}

func locationByID(locations []model.LocationState, locationID string) (model.LocationState, bool) {
	for _, location := range locations {
		if location.ID == locationID {
			return location, true
		}
	}
	return model.LocationState{}, false
}

func childLocations(locations []model.LocationState, parentID string) []model.LocationState {
	children := []model.LocationState{}
	for _, location := range locations {
		if location.ParentLocationID == parentID {
			children = append(children, location)
		}
	}
	return children
}

func siblingLocations(locations []model.LocationState, target model.LocationState) []model.LocationState {
	if target.ParentLocationID == "" {
		return nil
	}
	siblings := []model.LocationState{}
	for _, location := range locations {
		if location.ID != target.ID && location.ParentLocationID == target.ParentLocationID {
			siblings = append(siblings, location)
		}
	}
	return siblings
}

func ancestorLocations(locations []model.LocationState, location model.LocationState) []model.LocationState {
	byID := map[string]model.LocationState{}
	for _, candidate := range locations {
		byID[candidate.ID] = candidate
	}
	ancestors := []model.LocationState{}
	parentID := location.ParentLocationID
	for parentID != "" {
		parent, ok := byID[parentID]
		if !ok {
			break
		}
		ancestors = append(ancestors, parent)
		parentID = parent.ParentLocationID
	}
	for i, j := 0, len(ancestors)-1; i < j; i, j = i+1, j-1 {
		ancestors[i], ancestors[j] = ancestors[j], ancestors[i]
	}
	return ancestors
}

func visibleLocations(locations []model.LocationState) []model.LocationState {
	out := make([]model.LocationState, 0, len(locations))
	for _, location := range locations {
		out = append(out, visibleLocation(location))
	}
	return out
}

func areaByID(areas []model.MapArea, areaID string) model.MapArea {
	for _, area := range areas {
		if area.ID == areaID {
			return area
		}
	}
	return model.MapArea{}
}

func normalizeChildScale(parent model.LocationState, requested string) string {
	requested = strings.TrimSpace(requested)
	switch normalizedScale(parent) {
	case model.LocationScaleRegion:
		return model.LocationScaleSettlement
	case model.LocationScaleSettlement:
		if requested == model.LocationScaleDistrict || requested == model.LocationScaleSite {
			return requested
		}
		return model.LocationScaleSite
	case model.LocationScaleDistrict:
		if requested == model.LocationScaleRoom {
			return requested
		}
		return model.LocationScaleSite
	case model.LocationScaleSite:
		return model.LocationScaleRoom
	default:
		return ""
	}
}

func mergeLocations(existing []model.LocationState, updates []model.LocationState) []model.LocationState {
	byID := map[string]model.LocationState{}
	for _, location := range existing {
		byID[location.ID] = location
	}
	for _, location := range updates {
		byID[location.ID] = location
	}
	merged := make([]model.LocationState, 0, len(byID))
	for _, location := range byID {
		merged = append(merged, location)
	}
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].CreatedAt.Before(merged[j].CreatedAt) })
	return merged
}

func mergeStringAnyMaps(base map[string]any, overlay map[string]any) map[string]any {
	out := cloneMap(base)
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

func defaultLocationDescription(location model.LocationState) string {
	return strings.TrimSpace(fmt.Sprintf("%s is a %s-scale %s.", location.Name, normalizedScale(location), firstNonBlank(location.Type, "location")))
}

func defaultAffordances(location model.LocationState) []string {
	switch normalizedScale(location) {
	case model.LocationScaleSettlement:
		return []string{"enter", "leave", "ask around", "seek shelter", "look for a contact"}
	case model.LocationScaleDistrict:
		return []string{"observe movement", "find a specific site", "ask local people"}
	case model.LocationScaleSite:
		return []string{"enter carefully", "watch exits", "meet someone", "search for traces"}
	case model.LocationScaleRoom:
		return []string{"search", "hide", "speak quietly", "listen"}
	default:
		return []string{"observe", "move through"}
	}
}

func defaultRisks(location model.LocationState) []string {
	switch firstNonBlank(location.Type, normalizedScale(location)) {
	case "gate", "authority", "outpost":
		return []string{"guards", "questions", "restricted access"}
	case "market", "meeting_place":
		return []string{"crowds", "rumors", "being noticed"}
	case "alley", "ruin":
		return []string{"poor visibility", "ambush", "unreliable witnesses"}
	default:
		return []string{"being seen", "lost time"}
	}
}

func defaultResources(location model.LocationState) []string {
	switch firstNonBlank(location.Type, normalizedScale(location)) {
	case "gate":
		return []string{"entry control", "watch list", "guards"}
	case "market":
		return []string{"goods", "crowd cover", "rumors"}
	case "authority":
		return []string{"records", "orders", "local power"}
	case "inn":
		return []string{"rooms", "travelers", "private tables"}
	default:
		return []string{"local information"}
	}
}

func defaultAccessRules(location model.LocationState) []string {
	switch firstNonBlank(location.Type, normalizedScale(location)) {
	case "authority", "storehouse", "private_room":
		return []string{"requires reason or permission"}
	case "gate":
		return []string{"controlled by local order"}
	default:
		return []string{"public access unless story state says otherwise"}
	}
}

func firstStringSlice(value any, fallback []string) []string {
	switch typed := value.(type) {
	case []string:
		if len(typed) > 0 {
			return typed
		}
	case []any:
		out := []string{}
		for _, item := range typed {
			if text := asString(item); text != "" {
				out = append(out, text)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return fallback
}

func asString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func locationDistanceValue(a model.LocationState, b model.LocationState) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func factionInfluencesForLocation(influences []model.FactionInfluence, locationID string) []model.FactionInfluence {
	out := []model.FactionInfluence{}
	for _, influence := range influences {
		if influence.LocationID == locationID {
			out = append(out, influence)
		}
	}
	return out
}

var _ port.LocationInspectionService = (*LocationInspectionService)(nil)
