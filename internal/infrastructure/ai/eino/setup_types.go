package eino

type setupAgentOutput struct {
	AuthorBible          setupAuthorBibleOutput     `json:"author_bible"`
	WorldState           []setupWorldStateOutput    `json:"world_state"`
	Characters           []setupCharacterOutput     `json:"characters"`
	Relationships        []setupRelationshipOutput  `json:"relationships"`
	OpenQuestions        []setupQuestionOutput      `json:"open_questions"`
	AssistantSummary     string                     `json:"assistant_summary"`
	VisualDraft          setupVisualDraftOutput     `json:"visual_draft"`
	NextAgentSuggestions []setupNextAgentSuggestion `json:"next_agent_suggestions"`
}

type setupAuthorBibleOutput struct {
	Theme               string   `json:"theme"`
	StyleGuide          string   `json:"style_guide"`
	WorldRules          []string `json:"world_rules"`
	AestheticPrinciples []string `json:"aesthetic_principles"`
	HardConstraints     []string `json:"hard_constraints"`
	SoftPreferences     []string `json:"soft_preferences"`
	ForbiddenMoves      []string `json:"forbidden_moves"`
}

type setupWorldStateOutput struct {
	Key        string `json:"key"`
	Value      any    `json:"value"`
	Note       string `json:"note"`
	Importance int    `json:"importance"`
	Volatility int    `json:"volatility"`
}

type setupCharacterOutput struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Profile     string   `json:"profile"`
	Personality string   `json:"personality"`
	VoiceStyle  string   `json:"voice_style"`
	Goals       []string `json:"goals"`
	Fears       []string `json:"fears"`
	Secrets     []string `json:"secrets"`
	Constraints []string `json:"constraints"`
}

type setupRelationshipOutput struct {
	CharacterAKey  string                      `json:"character_a_key"`
	CharacterBKey  string                      `json:"character_b_key"`
	Summary        string                      `json:"summary"`
	Anchors        []string                    `json:"anchors"`
	TensionPoints  []string                    `json:"tension_points"`
	SharedHistory  []string                    `json:"shared_history"`
	Volatility     int                         `json:"volatility"`
	CharacterAView setupRelationshipViewOutput `json:"character_a_view"`
	CharacterBView setupRelationshipViewOutput `json:"character_b_view"`
}

type setupRelationshipViewOutput struct {
	PublicAttitude         string `json:"public_attitude"`
	PrivateAttitude        string `json:"private_attitude"`
	BelievedTargetAttitude string `json:"believed_target_attitude"`
	MaskingStrategy        string `json:"masking_strategy"`
}

type setupQuestionOutput struct {
	Key          string `json:"key"`
	Question     string `json:"question"`
	WhyItMatters string `json:"why_it_matters"`
}

type setupVisualDraftOutput struct {
	Logline              string                         `json:"logline"`
	StyleTags            []string                       `json:"style_tags"`
	Tone                 string                         `json:"tone"`
	BoldnessLevel        int                            `json:"boldness_level"`
	WorldPressureCards   []setupWorldPressureCardOutput `json:"world_pressure_cards"`
	CharacterCards       []setupCharacterCardOutput     `json:"character_cards"`
	RelationshipEdges    []setupRelationshipEdgeOutput  `json:"relationship_edges"`
	OpenQuestions        []setupQuestionOutput          `json:"open_questions"`
	AgentSummary         string                         `json:"agent_summary"`
	NextAgentSuggestions []setupNextAgentSuggestion     `json:"next_agent_suggestions"`
}

type setupWorldPressureCardOutput struct {
	Title                 string   `json:"title"`
	Detail                string   `json:"detail"`
	Stakes                string   `json:"stakes"`
	RelatedWorldStateKeys []string `json:"related_world_state_keys"`
}

type setupCharacterCardOutput struct {
	CharacterKey string `json:"character_key"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	Hook         string `json:"hook"`
	Goal         string `json:"goal"`
	Fear         string `json:"fear"`
	Secret       string `json:"secret"`
}

type setupRelationshipEdgeOutput struct {
	FromCharacterKey string `json:"from_character_key"`
	ToCharacterKey   string `json:"to_character_key"`
	Summary          string `json:"summary"`
	Tension          string `json:"tension"`
	Misreading       string `json:"misreading"`
}

type setupNextAgentSuggestion struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}
