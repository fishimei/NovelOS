package eino

type setupAgentOutput struct {
	AuthorBible      setupAuthorBibleOutput    `json:"author_bible"`
	WorldState       []setupWorldStateOutput   `json:"world_state"`
	Characters       []setupCharacterOutput    `json:"characters"`
	Relationships    []setupRelationshipOutput `json:"relationships"`
	OpenQuestions    []setupQuestionOutput     `json:"open_questions"`
	AssistantSummary string                    `json:"assistant_summary"`
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
