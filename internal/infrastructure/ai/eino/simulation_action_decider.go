package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	llmmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"

	"github.com/fishimei/NovelOS/internal/application/model"
	"github.com/fishimei/NovelOS/internal/application/port"
	"github.com/fishimei/NovelOS/internal/config"
)

type CharacterActionDecider struct {
	model             llmmodel.ToolCallingChatModel
	modelName         string
	prompt            string
	locationInspector port.LocationInspectionService
}

func NewCharacterActionDecider(ctx context.Context, cfg config.AIConfig, locationInspector port.LocationInspectionService) (*CharacterActionDecider, error) {
	chatModel, err := newOpenAIChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &CharacterActionDecider{model: chatModel, modelName: cfg.Model, prompt: cfg.StoryAgent.SimulationPrompt, locationInspector: locationInspector}, nil
}

func (d *CharacterActionDecider) Decide(ctx context.Context, input model.CharacterActionDecisionInput) (model.CharacterActionDecision, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	input = d.ensureReachableContext(ctx, input)
	inspected := inspectedLocationSet(input.InspectedLocations)
	return d.decideWithTools(ctx, input, inspected)
}

func (d *CharacterActionDecider) ensureReachableContext(ctx context.Context, input model.CharacterActionDecisionInput) model.CharacterActionDecisionInput {
	if d.locationInspector == nil || len(input.NearbyLocations) > 0 {
		return input
	}
	locationContext, err := d.locationInspector.EnsureReachableLocations(ctx, model.LocationReachabilityInput{
		ProjectID:         projectIDFromDecisionInput(input),
		CharacterID:       input.Character.ID,
		CurrentLocationID: input.Location.ID,
		World:             input.World,
	})
	if err != nil {
		return input
	}
	if locationContext.CurrentLocation.ID != "" {
		input.Location = locationContext.CurrentLocation
	}
	input.NearbyLocations = locationContext.ReachableLocations
	return input
}

func (d *CharacterActionDecider) decideWithTools(ctx context.Context, input model.CharacterActionDecisionInput, inspected map[string]struct{}) (model.CharacterActionDecision, error) {
	inspectTool, err := utils.InferTool("inspect_location", "检查当前角色可到达地点的细节。只在行动需要目标地点细节且输入尚未初始化该地点时调用。", func(ctx context.Context, request model.LocationInspectionInput) (model.LocationState, error) {
		return d.inspectActionLocation(ctx, input, request)
	})
	if err != nil {
		return model.CharacterActionDecision{}, err
	}
	submitTool, err := utils.InferTool("submit_character_action", "提交当前角色的下一步自主行动。最终必须调用此工具。", func(ctx context.Context, input model.CharacterActionDecision) (model.CharacterActionDecision, error) {
		return input, nil
	})
	if err != nil {
		return model.CharacterActionDecision{}, err
	}
	inspectInfo, err := inspectTool.Info(ctx)
	if err != nil {
		return model.CharacterActionDecision{}, err
	}
	submitInfo, err := submitTool.Info(ctx)
	if err != nil {
		return model.CharacterActionDecision{}, err
	}
	tools := []*schema.ToolInfo{inspectInfo, submitInfo}
	prompt := strings.TrimSpace(d.prompt) + "\n\n你可以调用 inspect_location 查看可到达地点的细节；最终必须调用 submit_character_action 提交行动。不要重复检查同一地点。"
	sharedPayload, _ := json.Marshal(buildActionDecisionSharedPrompt(input.World))
	characterPayload, _ := json.Marshal(buildActionDecisionCharacterPrompt(input))
	messages := []*schema.Message{
		schema.SystemMessage(prompt),
		schema.UserMessage("shared_context:\n" + string(sharedPayload)),
		schema.UserMessage("character_context:\n" + string(characterPayload)),
	}
	for step := 0; step < 5; step++ {
		msg, err := d.model.Generate(ctx, messages, toolCallOptions(d.modelName, 1200, tools)...)
		if err != nil {
			return model.CharacterActionDecision{}, err
		}
		if msg == nil || len(msg.ToolCalls) == 0 {
			messages = append(messages, msg, schema.UserMessage("必须调用 inspect_location 或 submit_character_action；如果已经能决定，立刻调用 submit_character_action。"))
			continue
		}
		messages = append(messages, msg)
		for _, call := range msg.ToolCalls {
			switch call.Function.Name {
			case "inspect_location":
				location, err := d.runInspectLocationTool(ctx, input, call.Function.Arguments)
				if err != nil {
					messages = append(messages, schema.ToolMessage(`{"ok":false,"error":`+quoteJSON(err.Error())+`}`, call.ID, schema.WithToolName(call.Function.Name)))
					continue
				}
				inspected[location.ID] = struct{}{}
				payload, _ := json.Marshal(map[string]any{"ok": true, "location": location})
				messages = append(messages, schema.ToolMessage(string(payload), call.ID, schema.WithToolName(call.Function.Name)))
			case "submit_character_action":
				var decision model.CharacterActionDecision
				if _, err := decodeModelJSONWithMeta(call.Function.Arguments, &decision); err != nil {
					messages = append(messages, schema.ToolMessage(`{"ok":false,"error":`+quoteJSON(err.Error())+`}`, call.ID, schema.WithToolName(call.Function.Name)))
					continue
				}
				decision = defaultDecisionTarget(decision, input.Location.ID)
				if err := validateCharacterActionDecision(input, decision, inspected); err != nil {
					messages = append(messages, schema.ToolMessage(`{"ok":false,"error":`+quoteJSON(err.Error())+`}`, call.ID, schema.WithToolName(call.Function.Name)))
					continue
				}
				return decision, nil
			default:
				messages = append(messages, schema.ToolMessage(`{"ok":false,"error":"unexpected tool"}`, call.ID, schema.WithToolName(call.Function.Name)))
			}
		}
	}
	return model.CharacterActionDecision{}, fmt.Errorf("character action agent did not submit a valid action")
}

func (d *CharacterActionDecider) runInspectLocationTool(ctx context.Context, input model.CharacterActionDecisionInput, raw string) (model.LocationState, error) {
	var request model.LocationInspectionInput
	if _, err := decodeModelJSONWithMeta(raw, &request); err != nil {
		return model.LocationState{}, err
	}
	return d.inspectActionLocation(ctx, input, request)
}

func (d *CharacterActionDecider) inspectActionLocation(ctx context.Context, input model.CharacterActionDecisionInput, request model.LocationInspectionInput) (model.LocationState, error) {
	if d.locationInspector == nil {
		return model.LocationState{}, fmt.Errorf("inspect_location is unavailable")
	}
	locationID := strings.TrimSpace(request.LocationID)
	if locationID == "" {
		return model.LocationState{}, fmt.Errorf("location_id is required")
	}
	if !targetIsReachable(input, locationID) {
		return model.LocationState{}, fmt.Errorf("location_id %q is not current_location.id or reachable_location_refs[].id", locationID)
	}
	request.ProjectID = firstText(request.ProjectID, projectIDFromDecisionInput(input))
	request.CharacterID = firstText(request.CharacterID, input.Character.ID)
	request.CurrentLocationID = firstText(request.CurrentLocationID, input.Location.ID)
	request.LocationID = locationID
	request.World = input.World
	result, err := d.locationInspector.InspectLocation(ctx, request)
	if err != nil {
		return model.LocationState{}, err
	}
	return result.InspectedLocation, nil
}

func forcedToolCall(msg *schema.Message, name string) (schema.ToolCall, error) {
	if msg == nil || len(msg.ToolCalls) == 0 {
		return schema.ToolCall{}, fmt.Errorf("character action agent did not call %s", name)
	}
	for _, call := range msg.ToolCalls {
		if call.Function.Name == name {
			return call, nil
		}
	}
	return schema.ToolCall{}, fmt.Errorf("character action agent called unexpected tool")
}

func (d *CharacterActionDecider) decideWithoutTools(ctx context.Context, input model.CharacterActionDecisionInput, inspected map[string]struct{}) (model.CharacterActionDecision, error) {
	fallbackPrompt := strings.TrimSpace(d.prompt) + "\n\n当前模型代理不兼容 tool calling。不要调用工具，直接输出严格 JSON 对象，不要 markdown，不要解释。"
	sharedPayload, _ := json.Marshal(buildActionDecisionSharedPrompt(input.World))
	characterPayload, _ := json.Marshal(buildActionDecisionCharacterPrompt(input))
	messages := []*schema.Message{
		schema.SystemMessage(fallbackPrompt),
		schema.UserMessage("shared_context:\n" + string(sharedPayload)),
		schema.UserMessage("character_context:\n" + string(characterPayload)),
	}
	msg, err := d.model.Generate(ctx, messages, maxTokensOption(d.modelName, 1200))
	if err != nil {
		return model.CharacterActionDecision{}, err
	}
	var decision model.CharacterActionDecision
	if _, err := decodeModelJSONWithMeta(msg.Content, &decision); err != nil {
		return model.CharacterActionDecision{}, err
	}
	decision = defaultDecisionTarget(decision, input.Location.ID)
	if err := validateCharacterActionDecision(input, decision, inspected); err != nil {
		return model.CharacterActionDecision{}, err
	}
	return decision, nil
}

func validateCharacterActionDecision(input model.CharacterActionDecisionInput, decision model.CharacterActionDecision, inspected map[string]struct{}) error {
	target := strings.TrimSpace(decision.TargetLocationKey)
	if target == "" {
		return fmt.Errorf("target_location_key is required")
	}
	if !targetIsReachable(input, target) {
		return fmt.Errorf("target_location_key %q is not current_location.id or reachable_location_refs[].id", target)
	}
	if target != input.Location.ID && actionNeedsInspectedTarget(decision) && !targetIsInspectedOrInitialized(input, target, inspected) {
		return fmt.Errorf("inspect_location is required before choosing target_location_key %q for this action", target)
	}
	if err := validateDecisionParticipants(input, decision.ParticipantIDs); err != nil {
		return err
	}
	if decision.DurationHours <= 0 {
		return fmt.Errorf("duration_hours must be positive")
	}
	return nil
}

func validateDecisionParticipants(input model.CharacterActionDecisionInput, participantIDs []string) error {
	if len(input.World.Characters) == 0 {
		return nil
	}
	for _, participantID := range participantIDs {
		participantID = strings.TrimSpace(participantID)
		if participantID == "" {
			continue
		}
		if _, ok := input.World.Characters[participantID]; !ok {
			return fmt.Errorf("participant_id %q is not a project character id", participantID)
		}
	}
	return nil
}

func targetIsReachable(input model.CharacterActionDecisionInput, target string) bool {
	if target != "" && target == input.Location.ID {
		return true
	}
	for _, nearby := range input.NearbyLocations {
		if nearby.Location.ID == target {
			return true
		}
	}
	return false
}

func actionNeedsInspectedTarget(decision model.CharacterActionDecision) bool {
	actionType := normalizeStoryActionType(decision.ActionType, "", decision.Description)
	return actionType != "observe" && actionType != "silence"
}

func targetIsInspectedOrInitialized(input model.CharacterActionDecisionInput, target string, inspected map[string]struct{}) bool {
	if _, ok := inspected[target]; ok {
		return true
	}
	for _, reachable := range input.NearbyLocations {
		if reachable.Location.ID == target && reachable.Location.DetailState == model.LocationDetailInitialized {
			return true
		}
	}
	return false
}

func defaultDecisionTarget(decision model.CharacterActionDecision, currentLocationID string) model.CharacterActionDecision {
	if strings.TrimSpace(decision.TargetLocationKey) == "" {
		decision.TargetLocationKey = currentLocationID
	}
	return decision
}

func inspectedLocationSet(locations []model.LocationState) map[string]struct{} {
	seen := map[string]struct{}{}
	for _, location := range locations {
		if location.ID != "" {
			seen[location.ID] = struct{}{}
		}
	}
	return seen
}

func projectIDFromDecisionInput(input model.CharacterActionDecisionInput) string {
	if input.Location.ProjectID != "" {
		return input.Location.ProjectID
	}
	if input.Character.ProjectID != "" {
		return input.Character.ProjectID
	}
	for _, location := range input.World.Locations {
		if location.ProjectID != "" {
			return location.ProjectID
		}
	}
	return ""
}

func quoteJSON(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
