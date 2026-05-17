package eino

import (
	"context"
	"testing"
)

func TestNewSetupToolsRegistersExpectedTools(t *testing.T) {
	tools, err := newSetupTools(&setupRunState{})
	if err != nil {
		t.Fatalf("newSetupTools returned error: %v", err)
	}
	if len(tools) != 3 {
		t.Fatalf("expected three setup tools, got %d", len(tools))
	}
}

func TestShowSetupDraftSavesDetailedDraft(t *testing.T) {
	state := &setupRunState{}
	out, err := showSetupDraft(context.Background(), state, detailedShowSetupDraftInput())
	if err != nil {
		t.Fatalf("showSetupDraft returned error: %v", err)
	}
	if len(out.Characters) != 2 {
		t.Fatalf("expected two characters, got %d", len(out.Characters))
	}
	if out.AuthorBible.Theme != "想象力与秩序的冲突" {
		t.Fatalf("unexpected theme %q", out.AuthorBible.Theme)
	}
	if out.VisualDraft.Logline == "" || len(out.VisualDraft.WorldPressureCards) != 1 || len(out.VisualDraft.CharacterCards) != 2 {
		t.Fatalf("unexpected visual draft: %#v", out.VisualDraft)
	}
	if len(out.VisualDraft.NextAgentSuggestions) != 1 {
		t.Fatalf("expected next agent suggestions to be mirrored into visual draft: %#v", out.VisualDraft.NextAgentSuggestions)
	}
	stored := state.agentOutput()
	if stored.AssistantSummary != out.AssistantSummary {
		t.Fatalf("state did not save output: %#v", stored)
	}
}

func TestReviseSetupDraftUsesRevisionSummaryFallback(t *testing.T) {
	state := &setupRunState{}
	out, err := reviseSetupDraft(context.Background(), state, ReviseSetupDraftInput{
		RevisionIntent:  "更未来、更大胆",
		RevisionSummary: "已改成更激进的未来都市设定。",
		AuthorBible: setupAuthorBibleOutput{
			Theme:      "记忆产权与自我意志",
			StyleGuide: "高概念、明亮、带危险感",
		},
		WorldState: []setupWorldStateOutput{{Key: "memory_market", Value: "梦境记忆可以交易", Importance: 5, Volatility: 4}},
		Characters: []setupCharacterOutput{{Key: "broker", Name: "栖迟", Role: "protagonist", Profile: "非法梦境经纪人。"}},
		VisualDraft: setupVisualDraftOutput{
			Logline:        "一座城市把梦卖给未来。",
			CharacterCards: []setupCharacterCardOutput{{CharacterKey: "broker", Name: "栖迟", Role: "protagonist", Hook: "她卖掉的梦正在回来追债。"}},
		},
	})
	if err != nil {
		t.Fatalf("reviseSetupDraft returned error: %v", err)
	}
	if out.AssistantSummary != "已改成更激进的未来都市设定。" {
		t.Fatalf("unexpected assistant summary %q", out.AssistantSummary)
	}
	if out.AuthorBible.Theme != "记忆产权与自我意志" {
		t.Fatalf("unexpected revised theme %q", out.AuthorBible.Theme)
	}
}

func TestHandoffNextAgentRecordsSuggestions(t *testing.T) {
	state := &setupRunState{}
	if _, err := showSetupDraft(context.Background(), state, detailedShowSetupDraftInput()); err != nil {
		t.Fatalf("showSetupDraft returned error: %v", err)
	}
	suggestions := []setupNextAgentSuggestion{{Key: "first_chapter", Label: "进入第一章编排", Reason: "设定张力已经足够启动首章。"}}
	out, err := handoffNextAgent(context.Background(), state, HandoffNextAgentInput{NextAgentSuggestions: suggestions})
	if err != nil {
		t.Fatalf("handoffNextAgent returned error: %v", err)
	}
	if len(out) != 1 || out[0].Key != "first_chapter" {
		t.Fatalf("unexpected handoff output: %#v", out)
	}
	stored := state.agentOutput()
	if len(stored.NextAgentSuggestions) != 1 || stored.VisualDraft.NextAgentSuggestions[0].Key != "first_chapter" {
		t.Fatalf("state did not record suggestions: %#v", stored)
	}
}

func TestShowSetupDraftRejectsMissingCharacters(t *testing.T) {
	_, err := showSetupDraft(context.Background(), &setupRunState{}, ShowSetupDraftInput{AssistantSummary: "没有角色。"})
	if err == nil {
		t.Fatal("expected missing characters error")
	}
}

func detailedShowSetupDraftInput() ShowSetupDraftInput {
	return ShowSetupDraftInput{
		AuthorBible: setupAuthorBibleOutput{
			Theme:               "想象力与秩序的冲突",
			StyleGuide:          "高概念、快节奏、具象奇观",
			WorldRules:          []string{"梦境基础设施由城市统一调度"},
			AestheticPrinciples: []string{"明亮未来感与失控梦境并置"},
			HardConstraints:     []string{"不把世界规则解释成万能魔法"},
			SoftPreferences:     []string{"保持天马行空但人物动机可落地"},
			ForbiddenMoves:      []string{"不要用失忆直接抹平冲突"},
		},
		WorldState: []setupWorldStateOutput{{Key: "dream_grid", Value: "全城共享梦网", Note: "公共梦境开始泄漏到白天街道。", Importance: 5, Volatility: 4}},
		Characters: []setupCharacterOutput{
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
		},
		Relationships: []setupRelationshipOutput{{
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
		}},
		OpenQuestions:    []setupQuestionOutput{{Key: "ending_scale", Question: "结局更偏城市级救赎还是个人代价？", WhyItMatters: "会影响梦网失控的最终规模。"}},
		AssistantSummary: "已形成未来大胆构思，重点是梦境公共设施失控与双主角互相试探。",
		VisualDraft: setupVisualDraftOutput{
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
			},
			RelationshipEdges: []setupRelationshipEdgeOutput{{FromCharacterKey: "architect", ToCharacterKey: "warden", Summary: "合作审判", Tension: "信任不足", Misreading: "彼此都以为对方只想控制局面"}},
			OpenQuestions:     []setupQuestionOutput{{Key: "ending_scale", Question: "结局更偏城市级救赎还是个人代价？", WhyItMatters: "会影响梦网失控的最终规模。"}},
			AgentSummary:      "这版强调高概念奇观与人物互相误判。",
		},
		NextAgentSuggestions: []setupNextAgentSuggestion{{Key: "relationship_deepening", Label: "进入关系深化", Reason: "双主角误判是首章推进核心。"}},
	}
}
