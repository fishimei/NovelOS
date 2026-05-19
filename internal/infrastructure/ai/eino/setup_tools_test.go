package eino

import (
	"context"
	"strings"
	"testing"
)

func TestNewSetupToolsRegistersExpectedTools(t *testing.T) {
	tools, err := newSetupTools(&setupRunState{})
	if err != nil {
		t.Fatalf("newSetupTools returned error: %v", err)
	}
	if len(tools) != 6 {
		t.Fatalf("expected six setup tools, got %d", len(tools))
	}
}

func TestSetupToolsAccumulateAndFinalizeDraft(t *testing.T) {
	ctx := context.Background()
	state := &setupRunState{}

	if _, err := setSetupAuthorBible(ctx, state, SetSetupAuthorBibleInput{AuthorBible: detailedSetupAuthorBible()}); err != nil {
		t.Fatalf("setSetupAuthorBible returned error: %v", err)
	}
	if _, err := setSetupWorldState(ctx, state, SetSetupWorldStateInput{WorldState: detailedSetupWorldState()}); err != nil {
		t.Fatalf("setSetupWorldState returned error: %v", err)
	}
	if _, err := setSetupCharacters(ctx, state, SetSetupCharactersInput{Characters: detailedSetupCharacters()}); err != nil {
		t.Fatalf("setSetupCharacters returned error: %v", err)
	}
	if _, err := setSetupRelationships(ctx, state, SetSetupRelationshipsInput{Relationships: detailedSetupRelationships()}); err != nil {
		t.Fatalf("setSetupRelationships returned error: %v", err)
	}
	if _, err := setSetupVisualDraft(ctx, state, SetSetupVisualDraftInput{
		OpenQuestions:        []setupQuestionOutput{{Key: "ending_scale", Question: "结局更偏城市级救赎还是个人代价？", WhyItMatters: "会影响梦网失控的最终规模。"}},
		AssistantSummary:     "已形成未来大胆构思，重点是梦境公共设施失控与三方互相试探。",
		VisualDraft:          detailedSetupVisualDraft(),
		NextAgentSuggestions: []setupNextAgentSuggestion{{Key: "relationship_deepening", Label: "进入关系深化", Reason: "三方误判是首章推进核心。"}},
	}); err != nil {
		t.Fatalf("setSetupVisualDraft returned error: %v", err)
	}

	out, err := finalizeSetupDraft(ctx, state, FinalizeSetupDraftInput{})
	if err != nil {
		t.Fatalf("finalizeSetupDraft returned error: %v", err)
	}
	if out.AuthorBible.Theme != "想象力与秩序的冲突" {
		t.Fatalf("unexpected theme %q", out.AuthorBible.Theme)
	}
	if len(out.WorldState) != 3 || len(out.Characters) != 3 || len(out.Relationships) != 2 {
		t.Fatalf("unexpected draft sizes: world=%d characters=%d relationships=%d", len(out.WorldState), len(out.Characters), len(out.Relationships))
	}
	if len(out.VisualDraft.OpenQuestions) != 1 {
		t.Fatalf("expected open questions to be mirrored into visual draft: %#v", out.VisualDraft.OpenQuestions)
	}
	if len(out.VisualDraft.NextAgentSuggestions) != 1 || out.VisualDraft.NextAgentSuggestions[0].Key != "relationship_deepening" {
		t.Fatalf("expected next agent suggestions to be mirrored into visual draft: %#v", out.VisualDraft.NextAgentSuggestions)
	}
	stored := state.agentOutput()
	if stored.AssistantSummary != out.AssistantSummary {
		t.Fatalf("state did not save output: %#v", stored)
	}
}

func TestFinalizeSetupDraftRejectsMissingSections(t *testing.T) {
	_, err := finalizeSetupDraft(context.Background(), &setupRunState{}, FinalizeSetupDraftInput{})
	if err == nil {
		t.Fatal("expected missing sections error")
	}
	if !strings.Contains(err.Error(), "author_bible") || !strings.Contains(err.Error(), "characters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func detailedSetupAuthorBible() setupAuthorBibleOutput {
	return setupAuthorBibleOutput{
		Theme:               "想象力与秩序的冲突",
		StyleGuide:          "高概念、快节奏、具象奇观",
		WorldRules:          []string{"梦境基础设施由城市统一调度"},
		AestheticPrinciples: []string{"明亮未来感与失控梦境并置"},
		HardConstraints:     []string{"不把世界规则解释成万能魔法"},
		SoftPreferences:     []string{"保持天马行空但人物动机可落地"},
		ForbiddenMoves:      []string{"不要用失忆直接抹平冲突"},
	}
}

func detailedSetupWorldState() []setupWorldStateOutput {
	return []setupWorldStateOutput{
		{Key: "dream_grid", Value: "全城共享梦网", Note: "公共梦境开始泄漏到白天街道。", Importance: 5, Volatility: 4},
		{Key: "memory_market", Value: "梦境记忆可以交易", Note: "黑市正在抢夺梦境碎片。", Importance: 4, Volatility: 3},
		{Key: "public_order", Value: "监管部门封锁事故区", Note: "官方压制消息导致民间猜疑升级。", Importance: 4, Volatility: 4},
	}
}

func detailedSetupCharacters() []setupCharacterOutput {
	return []setupCharacterOutput{
		{
			Key:         "architect",
			Name:        "鹿见",
			Role:        "protagonist",
			Profile:     "修复梦网的年轻工程师。",
			Personality: "锋利、好奇、抗拒权威",
			VoiceStyle:  "短句、跳跃比喻",
			Goals:       []string{"找出梦网失控源头"},
			Fears:       []string{"自己的童年记忆是伪造的"},
			Secrets:     []string{"她能在清醒时改写梦境"},
			Constraints: []string{"不能随意读取他人梦境"},
		},
		{
			Key:         "warden",
			Name:        "闻昼",
			Role:        "authority",
			Profile:     "负责监管梦网的城市执行官。",
			Personality: "克制、冷静、疑心重",
			VoiceStyle:  "礼貌但压迫感强",
			Goals:       []string{"阻止梦网崩溃"},
			Fears:       []string{"真相会摧毁城市合法性"},
			Secrets:     []string{"他曾授权一次非法梦境封存"},
			Constraints: []string{"必须维持公共秩序"},
		},
		{
			Key:         "broker",
			Name:        "栖迟",
			Role:        "wildcard",
			Profile:     "非法梦境经纪人。",
			Personality: "散漫、机敏、危险地诚实",
			VoiceStyle:  "像在讲笑话一样递刀",
			Goals:       []string{"卖出能改变城市命运的梦"},
			Fears:       []string{"被自己偷来的记忆吞没"},
			Secrets:     []string{"她知道鹿见童年的封存编号"},
			Constraints: []string{"不能暴露黑市客户名单"},
		},
	}
}

func detailedSetupRelationships() []setupRelationshipOutput {
	return []setupRelationshipOutput{
		{
			CharacterAKey: "architect",
			CharacterBKey: "warden",
			Summary:       "修复者与监管者互相需要也互相审判。",
			Anchors:       []string{"共同掌握梦网事故现场"},
			TensionPoints: []string{"她怀疑他隐瞒真相", "他担心她就是漏洞源头"},
			SharedHistory: []string{"三年前同一场梦网封存事故改变了两人的人生"},
			Volatility:    4,
			CharacterAView: setupRelationshipViewOutput{
				PublicAttitude:         "配合调查",
				PrivateAttitude:        "认为对方隐藏了梦网真相",
				BelievedTargetAttitude: "对方只把她当工具",
				MaskingStrategy:        "用技术细节遮掩试探",
			},
			CharacterBView: setupRelationshipViewOutput{
				PublicAttitude:         "提供权限",
				PrivateAttitude:        "担心她就是灾难源头",
				BelievedTargetAttitude: "她迟早会越界",
				MaskingStrategy:        "以制度语言压住私人恐惧",
			},
		},
		{
			CharacterAKey: "architect",
			CharacterBKey: "broker",
			Summary:       "工程师追查事故，经纪人出售真相碎片。",
			Anchors:       []string{"都握有封存事故的一部分证据"},
			TensionPoints: []string{"鹿见需要线索", "栖迟需要买家"},
			SharedHistory: []string{"两人的记忆都被同一批梦境碎片污染"},
			Volatility:    5,
			CharacterAView: setupRelationshipViewOutput{
				PublicAttitude:         "暂时交易",
				PrivateAttitude:        "不信任但被吸引",
				BelievedTargetAttitude: "对方只看重利益",
				MaskingStrategy:        "把好奇包装成审讯",
			},
			CharacterBView: setupRelationshipViewOutput{
				PublicAttitude:         "调侃合作",
				PrivateAttitude:        "想确认她是否是封存钥匙",
				BelievedTargetAttitude: "她迟早会来求真相",
				MaskingStrategy:        "用玩笑掩盖试探",
			},
		},
	}
}

func detailedSetupVisualDraft() setupVisualDraftOutput {
	return setupVisualDraftOutput{
		Logline:       "未来城市的公共梦网泄漏进现实，一个修梦工程师发现自己可能就是漏洞。",
		StyleTags:     []string{"未来都市", "高概念", "天马行空"},
		Tone:          "大胆明亮但有危险感",
		BoldnessLevel: 9,
		WorldPressureCards: []setupWorldPressureCardOutput{{
			Title:                 "梦网过载",
			Detail:                "公共梦境开始泄漏到白天街道。",
			Stakes:                "城市秩序和个人身份同时崩解。",
			RelatedWorldStateKeys: []string{"dream_grid"},
		}},
		CharacterCards: []setupCharacterCardOutput{
			{CharacterKey: "architect", Name: "鹿见", Role: "protagonist", Hook: "能修梦，也可能是漏洞本身。", Goal: "找出梦网失控源头", Fear: "自己的童年记忆是伪造的", Secret: "她能在清醒时改写梦境"},
			{CharacterKey: "warden", Name: "闻昼", Role: "authority", Hook: "守秩序的人可能亲手埋下灾难。", Goal: "阻止梦网崩溃", Fear: "真相摧毁城市合法性", Secret: "他曾授权非法梦境封存"},
			{CharacterKey: "broker", Name: "栖迟", Role: "wildcard", Hook: "卖梦的人握着主角失落的童年。", Goal: "卖出关键梦境", Fear: "被记忆吞没", Secret: "她知道封存编号"},
		},
		RelationshipEdges: []setupRelationshipEdgeOutput{
			{FromCharacterKey: "architect", ToCharacterKey: "warden", Summary: "合作审判", Tension: "信任不足", Misreading: "彼此都以为对方只想控制局面"},
			{FromCharacterKey: "architect", ToCharacterKey: "broker", Summary: "交易真相", Tension: "利益与好奇纠缠", Misreading: "彼此低估了对方的伤口"},
		},
		AgentSummary: "这版强调高概念奇观与人物互相误判。",
	}
}
