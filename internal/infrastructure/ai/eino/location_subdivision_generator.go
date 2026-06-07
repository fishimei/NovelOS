package eino

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
	"github.com/fishimei/NovelOS/internal/pkgerr"
)

type LocationSubdivisionGenerator struct {
	model     llmmodel.ToolCallingChatModel
	modelName string
}

func NewLocationSubdivisionGenerator(ctx context.Context, cfg config.AIConfig) (*LocationSubdivisionGenerator, error) {
	chatModel, err := newOpenAIChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &LocationSubdivisionGenerator{model: chatModel, modelName: cfg.Model}, nil
}

func (g *LocationSubdivisionGenerator) GenerateLocationSubdivision(ctx context.Context, input model.LocationSubdivisionInput) (model.LocationSubdivisionPlan, error) {
	if g == nil || g.model == nil {
		return model.LocationSubdivisionPlan{}, pkgerr.Internal("location subdivision model is not configured", nil)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	payload, _ := json.Marshal(locationSubdivisionPromptInput(input))
	msg, err := g.model.Generate(ctx, []*schema.Message{
		schema.SystemMessage(locationSubdivisionSystemPrompt()),
		schema.UserMessage(string(payload)),
	}, maxTokensOption(g.modelName, 1400))
	if err != nil {
		return model.LocationSubdivisionPlan{}, err
	}
	var plan model.LocationSubdivisionPlan
	if err := decodeModelJSON(msg.Content, &plan); err != nil {
		return model.LocationSubdivisionPlan{}, err
	}
	return plan, nil
}

func locationSubdivisionSystemPrompt() string {
	return strings.TrimSpace(`You generate hierarchical locations for a fiction world map.
Return exactly one JSON object with:
{
  "detail": {"name":"","type":"","description":"","properties":{"public_summary":"","affordances":[],"risks":[],"resources":[],"access_rules":[]}},
  "children": [{"name":"","type":"","scale":"","description":"","dx":0,"dy":0,"radius":1,"properties":{"public_summary":"","route_hint":""}}]
}
Rules:
- Do not invent ids, project ids, parent ids, timestamps, or status.
- Use the parent location, area terrain, sibling locations, factions, and world state.
- If need_children is false, children may be empty.
- Child scale must be appropriate for the parent hierarchy.
- Output JSON only.`)
}

func locationSubdivisionPromptInput(input model.LocationSubdivisionInput) map[string]any {
	return map[string]any{
		"project_id":         input.ProjectID,
		"parent_location":   input.ParentLocation,
		"area":              input.Area,
		"existing_children":  firstEntries(input.ExistingChildren, 12),
		"sibling_locations":  firstEntries(input.SiblingLocations, 12),
		"faction_influences": firstEntries(input.FactionInfluences, 8),
		"world_state":        input.World.WorldState,
		"reason":             input.Reason,
		"need_children":      input.NeedChildren,
	}
}

var _ port.LocationSubdivisionGenerator = (*LocationSubdivisionGenerator)(nil)
