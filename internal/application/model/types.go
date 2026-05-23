// Package model 瀹氫箟浜嗗簲鐢ㄧ▼搴忕殑鏍稿績棰嗗煙妯″瀷銆?
// 杩欎簺妯″瀷鏄笟鍔￠€昏緫鐨勬牳蹇冭〃绀猴紝鐙珛浜庝换浣曠壒瀹氱殑鎸佷箙鍖栨垨浼犺緭鏈哄埗銆?
// 妯″瀷鍒嗕负浠ヤ笅鍑犱釜涓昏绫诲埆锛?
// 1. 椤圭洰绠＄悊锛歅roject, ProjectDetail
// 2. 涓栫晫瑙傝瀹氾細AuthorBible, WorldStateEntry
// 3. 瑙掕壊绯荤粺锛欳haracter, CreateCharacterInput, UpdateCharacterInput
// 4. 鍏崇郴绯荤粺锛歊elationship, RelationshipPair, RelationshipView, RelationshipEvent
// 5. 璁剧疆娴佺▼锛歋etupSession, SetupRun, SetupDraft, SetupQuestion
// 6. 鏁呬簨娴佺▼锛歋torySession, StoryRun, Draft, PlotVariable, ReviewReport, MemoryPatch
// 7. 绔犺妭涓庤蹇嗭細Chapter, Memory
package model

import "time"

// PageInput 瀹氫箟鍒嗛〉鏌ヨ鐨勮緭鍏ュ弬鏁般€?
type PageInput struct {
	Page     int // 褰撳墠椤电爜锛屼粠 1 寮€濮?
	PageSize int // 姣忛〉鏉℃暟
}

// ListResult 鏄€氱敤鍒嗛〉缁撴灉鍖呰鍣ㄣ€?
// 鐢ㄤ簬杩斿洖甯︽湁鎬绘暟缁熻鐨勫垎椤垫暟鎹€?
type ListResult[T any] struct {
	Items []T // 褰撳墠椤电殑鏁版嵁椤?
	Total int // 绗﹀悎鏌ヨ鏉′欢鐨勬€绘潯鏁?
}

// CreateProjectInput 鏄垱寤烘柊椤圭洰鐨勮緭鍏ュ弬鏁般€?
type CreateProjectInput struct {
	Title       string // 椤圭洰鏍囬
	Genre       string // 浣滃搧绫诲瀷/浣撹
	Description string // 椤圭洰鎻忚堪
}

// UpdateProjectInput 鏄洿鏂伴」鐩俊鎭殑杈撳叆鍙傛暟銆?
type UpdateProjectInput struct {
	Title       string // 椤圭洰鏍囬
	Genre       string // 浣滃搧绫诲瀷/浣撹
	Description string // 椤圭洰鎻忚堪
}

// Project 鏄皬璇寸殑鍩烘湰椤圭洰鍗曞厓銆?
type Project struct {
	ID          string    `json:"id"`          // 椤圭洰鍞竴鏍囪瘑绗?
	Title       string    `json:"title"`       // 椤圭洰鏍囬
	Genre       string    `json:"genre"`       // 浣滃搧绫诲瀷/浣撹
	Description string    `json:"description"` // 椤圭洰鎻忚堪
	CreatedAt   time.Time `json:"created_at"`  // 鍒涘缓鏃堕棿
	UpdatedAt   time.Time `json:"updated_at"`  // 鏈€鍚庢洿鏂版椂闂?
}

// ProjectDetail 鏄」鐩殑璇︾粏淇℃伅锛屽寘鍚叧鑱斿疄浣撶殑缁熻銆?
type ProjectDetail struct {
	Project
	CharacterCount             int `json:"character_count"`               // 椤圭洰涓殑瑙掕壊鏁伴噺
	RelationshipCount          int `json:"relationship_count"`            // 椤圭洰涓殑鍏崇郴鏁伴噺
	StorySessionCount          int `json:"story_session_count"`           // 椤圭洰涓殑鏁呬簨浼氳瘽鏁伴噺
	LastCommittedChapterNumber int `json:"last_committed_chapter_number"` // 鏈€鍚庝竴涓凡鎻愪氦绔犺妭鐨勭紪鍙?
}

// WorldStateEntry 浠ｈ〃涓栫晫鐘舵€佷腑鐨勪竴涓潯鐩紝鐢ㄤ簬璺熻釜鏁呬簨涓栫晫涓殑鍏抽敭淇℃伅銆?
// 涓栫晫鐘舵€佹槸 AI 鍦ㄧ敓鎴愬唴瀹规椂闇€瑕佸弬鑰冪殑鑳屾櫙鐭ヨ瘑搴撱€?
type WorldStateEntry struct {
	ID         string    `json:"id"`         // 鏉＄洰鍞竴鏍囪瘑绗?
	ProjectID  string    `json:"project_id"` // 鎵€灞為」鐩?ID
	Key        string    `json:"key"`        // 鐘舵€侀敭锛堝 "king_name", "current_year"锛?
	Value      any       `json:"value"`      // 鐘舵€佸€硷紙鍙互鏄换鎰忕被鍨嬶級
	Note       string    `json:"note"`       // 澶囨敞璇存槑
	Status     string    `json:"status"`     // 鐘舵€佹爣璇?
	Importance int       `json:"importance"` // 閲嶈鎬х瓑绾э紙褰卞搷 AI 鐢熸垚鏃剁殑鏉冮噸锛?
	Volatility int       `json:"volatility"` // 鍙樺寲棰戠巼锛堥珮 volatility 琛ㄧず璇ョ姸鎬佸彲鑳介绻佸彉鍖栵級
	UpdatedAt  time.Time `json:"updated_at"` // 鏈€鍚庢洿鏂版椂闂?
}

// UpdateAuthorBibleInput 鏄洿鏂颁綔鑰呭湥缁忕殑杈撳叆鍙傛暟銆?
// 浣滆€呭湥缁忔槸鎸囧 AI 鐢熸垚椋庢牸鍜岃鍒欑殑鍏冩暟鎹泦鍚堛€?
type UpdateAuthorBibleInput struct {
	Theme               string            // 浣滃搧涓婚
	StyleGuide          string            // 椋庢牸鎸囧崡
	WorldRules          []string          // 涓栫晫瑙勫垯鍒楄〃
	AestheticPrinciples []string          // 瀹＄編鍘熷垯鍒楄〃
	HardConstraints     []string          // 纭€х害鏉燂紙蹇呴』閬靛畧鐨勮鍒欙級
	SoftPreferences     []string          // 杞€у亸濂斤紙灏介噺閬靛畧鐨勮鍒欙級
	ForbiddenMoves      []string          // 绂佹琛屼负
	InitialWorldState   []WorldStateEntry // 鍒濆涓栫晫鐘舵€?
}

// AuthorBible 鏄綔鑰呭湥缁忕殑瀹屾暣妯″瀷锛屽寘鍚寚瀵?AI 鍒涗綔鐨勫厓鏁版嵁銆?
type AuthorBible struct {
	ID                  string            `json:"id"`                   // 鍦ｇ粡鍞竴鏍囪瘑绗?
	ProjectID           string            `json:"project_id"`           // 鎵€灞為」鐩?ID
	Theme               string            `json:"theme"`                // 浣滃搧涓婚
	StyleGuide          string            `json:"style_guide"`          // 椋庢牸鎸囧崡
	WorldRules          []string          `json:"world_rules"`          // 涓栫晫瑙勫垯鍒楄〃
	AestheticPrinciples []string          `json:"aesthetic_principles"` // 瀹＄編鍘熷垯鍒楄〃
	HardConstraints     []string          `json:"hard_constraints"`     // 纭€х害鏉?
	SoftPreferences     []string          `json:"soft_preferences"`     // 杞€у亸濂?
	ForbiddenMoves      []string          `json:"forbidden_moves"`      // 绂佹琛屼负
	InitialWorldState   []WorldStateEntry `json:"initial_world_state"`  // 鍒濆涓栫晫鐘舵€?
	Status              string            `json:"status"`               // 鐘舵€佹爣璇?
	UpdatedAt           time.Time         `json:"updated_at"`           // 鏈€鍚庢洿鏂版椂闂?
}

// CreateCharacterInput 鏄垱寤烘柊瑙掕壊鐨勮緭鍏ュ弬鏁般€?
type CreateCharacterInput struct {
	Name        string   // 瑙掕壊鍚嶇О
	Role        string   // 瑙掕壊瀹氫綅锛堝 protagonist, antagonist, supporting锛?
	Profile     string   // 瑙掕壊绠€浠?
	Personality string   // 鎬ф牸鐗瑰緛鎻忚堪
	VoiceStyle  string   // 璇磋瘽椋庢牸
	Goals       []string // 瑙掕壊鐩爣鍒楄〃
	Fears       []string // 瑙掕壊鎭愭儳鍒楄〃
	Secrets     []string // 瑙掕壊绉樺瘑鍒楄〃
	Constraints []string // 瑙掕壊琛屼负绾︽潫鍒楄〃
}

// UpdateCharacterInput 鏄洿鏂拌鑹蹭俊鎭殑杈撳叆鍙傛暟銆?
type UpdateCharacterInput struct {
	Name        string
	Role        string
	Profile     string
	Personality string
	VoiceStyle  string
	Goals       []string
	Fears       []string
	Secrets     []string
	Constraints []string
}

// Character 鏄晠浜嬩腑鐨勮鑹插疄浣撱€?
// 瑙掕壊鏄晠浜嬬敓鎴愮殑鏍稿績瑕佺礌锛孉I 浼氭牴鎹鑹茬殑鐩爣銆佹亹鎯у拰绾︽潫鏉ョ敓鎴愮鍚堣鑹茬壒鐐圭殑瀵硅瘽鍜岃涓恒€?
type Character struct {
	ID                  string    `json:"id"`                    // 瑙掕壊鍞竴鏍囪瘑绗?
	ProjectID           string    `json:"project_id"`            // 鎵€灞為」鐩?ID
	Name                string    `json:"name"`                  // 瑙掕壊鍚嶇О
	Role                string    `json:"role"`                  // 瑙掕壊瀹氫綅
	Profile             string    `json:"profile"`               // 瑙掕壊绠€浠?
	Personality         string    `json:"personality"`           // 鎬ф牸鐗瑰緛鎻忚堪
	VoiceStyle          string    `json:"voice_style"`           // 璇磋瘽椋庢牸
	Goals               []string  `json:"goals"`                 // 瑙掕壊鐩爣鍒楄〃
	Fears               []string  `json:"fears"`                 // 瑙掕壊鎭愭儳鍒楄〃
	Secrets             []string  `json:"secrets"`               // 瑙掕壊绉樺瘑鍒楄〃
	Constraints         []string  `json:"constraints"`           // 瑙掕壊琛屼负绾︽潫鍒楄〃
	RecentMemorySummary string    `json:"recent_memory_summary"` // 杩戞湡璁板繂鎽樿锛堢敤浜?AI 鐢熸垚鏃剁殑涓婁笅鏂囷級
	Status              string    `json:"status"`                // 鐘舵€佹爣璇?
	CreatedAt           time.Time `json:"created_at"`            // 鍒涘缓鏃堕棿
	UpdatedAt           time.Time `json:"updated_at"`            // 鏈€鍚庢洿鏂版椂闂?
}

// CreateRelationshipInput 鏄垱寤鸿鑹插叧绯荤殑杈撳叆鍙傛暟銆?
// 鍏崇郴鎻忚堪浜嗕袱涓鑹蹭箣闂寸殑浜掑姩妯″紡鍜屽巻鍙层€?
type CreateRelationshipInput struct {
	CharacterAID   string                // 瑙掕壊 A 鐨?ID
	CharacterBID   string                // 瑙掕壊 B 鐨?ID
	Summary        string                // 鍏崇郴姒傝鎻忚堪
	Anchors        []string              // 鍏崇郴閿氱偣锛堝叧绯讳腑鐨勫叧閿簨浠讹級
	TensionPoints  []string              // 绱у紶鐐癸紙鍙兘瀵艰嚧鍐茬獊鐨勫洜绱狅級
	SharedHistory  []string              // 鍏卞悓缁忓巻
	Volatility     int                   // 鍏崇郴鍙樺寲绋嬪害
	CharacterAView RelationshipViewInput // 瑙掕壊 A 瀵瑰叧绯荤殑鐪嬫硶
	CharacterBView RelationshipViewInput // 瑙掕壊 B 瀵瑰叧绯荤殑鐪嬫硶
}

// UpdateRelationshipInput 鏄洿鏂板叧绯讳俊鎭殑杈撳叆鍙傛暟銆?
type UpdateRelationshipInput struct {
	Summary       string   // 鍏崇郴姒傝鎻忚堪
	Anchors       []string // 鍏崇郴閿氱偣
	TensionPoints []string // 绱у紶鐐?
	SharedHistory []string // 鍏卞悓缁忓巻
	Volatility    int      // 鍏崇郴鍙樺寲绋嬪害
}

// RelationshipViewInput 鏄叧绯昏瑙掔殑杈撳叆鍙傛暟锛屾弿杩板崟鏂瑰鍏崇郴鐨勮鐭ャ€?
type RelationshipViewInput struct {
	PublicAttitude         string // 鍏紑鎬佸害锛堜粬浜哄彲瑙佺殑鎬佸害锛?
	PrivateAttitude        string // 绉佷笅鎬佸害锛堢湡瀹炴€佸害锛?
	BelievedTargetAttitude string // 浠ヤ负瀵规柟鐨勬€佸害
	MaskingStrategy        string // 鎺╅グ绛栫暐
}

// RelationshipPair 浠ｈ〃涓や釜瑙掕壊涔嬮棿鐨勫叧绯诲銆?
// 鍏崇郴瀵规槸鍏崇郴鐨勫熀纭€瀹炰綋锛屽瓨鍌ㄥ弻鏂瑰叡鍚岀殑淇℃伅銆?
type RelationshipPair struct {
	ID               string    `json:"id"`                 // 鍏崇郴瀵瑰敮涓€鏍囪瘑绗?
	ProjectID        string    `json:"project_id"`         // 鎵€灞為」鐩?ID
	LeftCharacterID  string    `json:"left_character_id"`  // 宸︿晶瑙掕壊 ID锛堣鑹?A锛?
	RightCharacterID string    `json:"right_character_id"` // 鍙充晶瑙掕壊 ID锛堣鑹?B锛?
	Summary          string    `json:"summary"`            // 鍏崇郴姒傝鎻忚堪
	Anchors          []string  `json:"anchors"`            // 鍏崇郴閿氱偣
	TensionPoints    []string  `json:"tension_points"`     // 绱у紶鐐?
	SharedHistory    []string  `json:"shared_history"`     // 鍏卞悓缁忓巻
	Volatility       int       `json:"volatility"`         // 鍏崇郴鍙樺寲绋嬪害
	Status           string    `json:"status"`             // 鐘舵€佹爣璇?
	CreatedAt        time.Time `json:"created_at"`         // 鍒涘缓鏃堕棿
	UpdatedAt        time.Time `json:"updated_at"`         // 鏈€鍚庢洿鏂版椂闂?
}

// RelationshipView 浠ｈ〃瑙掕壊瀵瑰叧绯荤殑鍗曟柟瑙嗚銆?
// 姣忎釜 RelationshipPair 浼氭湁涓や釜 RelationshipView锛屽垎鍒弿杩颁袱涓鑹插杩欐鍏崇郴鐨勮鐭ャ€?
type RelationshipView struct {
	ID                     string    `json:"id"`                       // 瑙嗚鍞竴鏍囪瘑绗?
	ProjectID              string    `json:"project_id"`               // 鎵€灞為」鐩?ID
	PairID                 string    `json:"pair_id"`                  // 鎵€灞炲叧绯诲 ID
	SourceCharacterID      string    `json:"source_character_id"`      // 瑙嗚鏉ユ簮瑙掕壊 ID
	TargetCharacterID      string    `json:"target_character_id"`      // 瑙嗚鐩爣瑙掕壊 ID
	PublicAttitude         string    `json:"public_attitude"`          // 鍏紑鎬佸害
	PrivateAttitude        string    `json:"private_attitude"`         // 绉佷笅鎬佸害
	BelievedTargetAttitude string    `json:"believed_target_attitude"` // 浠ヤ负瀵规柟鐨勬€佸害
	MaskingStrategy        string    `json:"masking_strategy"`         // 鎺╅グ绛栫暐
	Status                 string    `json:"status"`                   // 鐘舵€佹爣璇?
	CreatedAt              time.Time `json:"created_at"`               // 鍒涘缓鏃堕棿
	UpdatedAt              time.Time `json:"updated_at"`               // 鏈€鍚庢洿鏂版椂闂?
}

// RelationshipEvent 浠ｈ〃鍏崇郴涓彂鐢熺殑浜嬩欢锛岀敤浜庤窡韪叧绯荤殑鍙樺寲鍘嗗彶銆?
type RelationshipEvent struct {
	ID        string         `json:"id"`         // 浜嬩欢鍞竴鏍囪瘑绗?
	ProjectID string         `json:"project_id"` // 鎵€灞為」鐩?ID
	PairID    string         `json:"pair_id"`    // 鎵€灞炲叧绯诲 ID
	EventType string         `json:"event_type"` // 浜嬩欢绫诲瀷
	Summary   string         `json:"summary"`    // 浜嬩欢鎽樿
	Payload   map[string]any `json:"payload"`    // 浜嬩欢闄勫姞鏁版嵁
	CreatedAt time.Time      `json:"created_at"` // 浜嬩欢鍙戠敓鏃堕棿
}

// Relationship 鏄叧绯荤殑瀹屾暣鑱氬悎锛屽寘鍚叧绯诲銆佸弻鏂硅瑙掑拰鏈€杩戜簨浠躲€?
type Relationship struct {
	Pair           RelationshipPair    `json:"pair"`             // 鍏崇郴瀵瑰熀纭€淇℃伅
	Views          []RelationshipView  `json:"views"`            // 鎵€鏈夌浉鍏宠瑙?
	RecentEvents   []RelationshipEvent `json:"recent_events"`    // 鏈€杩戝彂鐢熺殑浜嬩欢
	CharacterAView *RelationshipView   `json:"character_a_view"` // 瑙掕壊 A 鐨勮瑙?
	CharacterBView *RelationshipView   `json:"character_b_view"` // 瑙掕壊 B 鐨勮瑙?
}

// CreateSetupSessionInput 鏄垱寤鸿缃細璇濈殑杈撳叆鍙傛暟銆?
// 璁剧疆浼氳瘽鐢ㄤ簬閫氳繃 AI 杈呭姪灏嗙矖鐣ユ兂娉曡浆鍖栦负缁撴瀯鍖栭」鐩姸鎬併€?
type CreateSetupSessionInput struct {
	SeedIdea string // 绉嶅瓙鎯虫硶/鍒濆姒傚康
}

// AdvanceSetupSessionInput 鏄帹杩涜缃細璇濈殑杈撳叆鍙傛暟銆?
type AdvanceSetupSessionInput struct {
	UserMessage string // 鐢ㄦ埛杈撳叆鐨勬秷鎭?
}

// ApplySetupRunInput 鏄簲鐢ㄨ缃繍琛岀粨鏋滅殑杈撳叆鍙傛暟銆?
// 鐢ㄦ埛鍙互閫夋嫨鎬у湴鎺ュ彈鐢熸垚鐨勫悇椤瑰唴瀹广€?
type ApplySetupRunInput struct {
	RunID               string `json:"run_id"`               // 瑕佸簲鐢ㄧ殑杩愯 ID
	AcceptAuthorBible   bool   `json:"accept_author_bible"`  // 鏄惁鎺ュ彈浣滆€呭湥缁?
	AcceptCharacters    bool   `json:"accept_characters"`    // 鏄惁鎺ュ彈瑙掕壊
	AcceptRelationships bool   `json:"accept_relationships"` // 鏄惁鎺ュ彈鍏崇郴
	AcceptWorldState    bool   `json:"accept_world_state"`   // 鏄惁鎺ュ彈涓栫晫鐘舵€?
	AuthorNote          string `json:"author_note"`          // 浣滆€呭娉?
}

// SetupQuestion 鏄缃繃绋嬩腑 AI 鎻愬嚭鐨勯棶棰樸€?
type SetupQuestion struct {
	Key          string `json:"key"`            // 闂鏍囪瘑閿?
	Question     string `json:"question"`       // 闂鍐呭
	WhyItMatters string `json:"why_it_matters"` // 涓轰粈涔堣繖涓棶棰橀噸瑕?
}

// ConversationMessage 鏄細璇濇秷鎭殑妯″瀷銆?
type ConversationMessage struct {
	ID        string    `json:"id"`         // 娑堟伅鍞竴鏍囪瘑绗?
	SessionID string    `json:"session_id"` // 鎵€灞炰細璇?ID
	Role      string    `json:"role"`       // 娑堟伅瑙掕壊锛坲ser/assistant锛?
	Content   string    `json:"content"`    // 娑堟伅鍐呭
	CreatedAt time.Time `json:"created_at"` // 鍒涘缓鏃堕棿
}

// SetupSession 鏄缃細璇濈殑妯″瀷銆?
// 璁剧疆浼氳瘽鏄綔鑰呬笌 AI 涔嬮棿鐨勫杞璇濓紝鐢ㄤ簬閫愭鏋勫缓椤圭洰鐨勫熀纭€璁惧畾銆?
type SetupSession struct {
	ID              string                `json:"id"`                          // 浼氳瘽鍞竴鏍囪瘑绗?
	ProjectID       string                `json:"project_id"`                  // 鎵€灞為」鐩?ID
	SeedIdea        string                `json:"seed_idea"`                   // 绉嶅瓙鎯虫硶
	LastUserMessage string                `json:"last_user_message"`           // 鏈€鍚庝竴鏉＄敤鎴锋秷鎭?
	Status          string                `json:"status"`                      // 浼氳瘽鐘舵€?
	LatestRunID     string                `json:"latest_run_id,omitempty"`     // 鏈€杩戜竴娆¤繍琛?ID
	LatestRunStatus string                `json:"latest_run_status,omitempty"` // 鏈€杩戜竴娆¤繍琛岀姸鎬?
	LatestRunError  string                `json:"latest_run_error,omitempty"`  // 鏈€杩戜竴娆¤繍琛岄敊璇?
	Messages        []ConversationMessage `json:"messages"`                    // 浼氳瘽娑堟伅鍘嗗彶
	CreatedAt       time.Time             `json:"created_at"`                  // 鍒涘缓鏃堕棿
	UpdatedAt       time.Time             `json:"updated_at"`                  // 鏈€鍚庢洿鏂版椂闂?
}

// SetupRun 鏄缃繍琛岀殑妯″瀷銆?
// 姣忔鐢ㄦ埛鎺ㄨ繘璁剧疆浼氳瘽鏃跺垱寤轰竴涓繍琛岋紝璺熻釜 AI 澶勭悊杩涘害銆?
type SetupRun struct {
	RunID       string    `json:"run_id"`       // 杩愯鍞竴鏍囪瘑绗?
	SessionID   string    `json:"session_id"`   // 鎵€灞炰細璇?ID
	ProjectID   string    `json:"project_id"`   // 鎵€灞為」鐩?ID
	Status      string    `json:"status"`       // 杩愯鐘舵€?
	CurrentStep string    `json:"current_step"` // 褰撳墠姝ラ
	Progress    int       `json:"progress"`     // 杩涘害鐧惧垎姣?
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"` // 鍒涘缓鏃堕棿
	UpdatedAt   time.Time `json:"updated_at"` // 鏈€鍚庢洿鏂版椂闂?
}

// SetupVisualWorldPressureCard 鏄?Setup 鑽夋涓敤浜庡睍绀轰笘鐣屽帇鍔涚殑鍗＄墖銆?
type SetupVisualWorldPressureCard struct {
	Title                 string   `json:"title"`
	Detail                string   `json:"detail"`
	Stakes                string   `json:"stakes"`
	RelatedWorldStateKeys []string `json:"related_world_state_keys"`
}

// SetupVisualCharacterCard 鏄?Setup 鑽夋涓敤浜庡睍绀鸿鑹插姛鑳戒綅鐨勫崱鐗囥€?
type SetupVisualCharacterCard struct {
	CharacterKey string `json:"character_key"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	Hook         string `json:"hook"`
	Goal         string `json:"goal"`
	Fear         string `json:"fear"`
	Secret       string `json:"secret"`
}

// SetupVisualRelationshipEdge 鏄?Setup 鑽夋涓敤浜庡睍绀哄叧绯诲浘鐨勮竟銆?
type SetupVisualRelationshipEdge struct {
	FromCharacterKey string `json:"from_character_key"`
	ToCharacterKey   string `json:"to_character_key"`
	Summary          string `json:"summary"`
	Tension          string `json:"tension"`
	Misreading       string `json:"misreading"`
}

// SetupNextAgentSuggestion 鏄?Setup 瀹屾垚鍚庡缓璁繘鍏ョ殑涓嬩竴姝?agent銆?
type SetupNextAgentSuggestion struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

// SetupVisualDraft 鏄潰鍚戠敤鎴风‘璁ょ殑鍙鍖?Setup 鑽夋瑙嗗浘銆?
type SetupVisualDraft struct {
	Logline              string                         `json:"logline"`
	StyleTags            []string                       `json:"style_tags"`
	Tone                 string                         `json:"tone"`
	BoldnessLevel        int                            `json:"boldness_level"`
	WorldPressureCards   []SetupVisualWorldPressureCard `json:"world_pressure_cards"`
	CharacterCards       []SetupVisualCharacterCard     `json:"character_cards"`
	RelationshipEdges    []SetupVisualRelationshipEdge  `json:"relationship_edges"`
	OpenQuestions        []SetupQuestion                `json:"open_questions"`
	AgentSummary         string                         `json:"agent_summary"`
	NextAgentSuggestions []SetupNextAgentSuggestion     `json:"next_agent_suggestions,omitempty"`
}

// SetupDraft 鏄缃敓鎴愮殑鑽夌銆?
// 鍖呭惈 AI 鐢熸垚鐨勫畬鏁撮」鐩瀹氥€?
type SetupDraft struct {
	AuthorBible      AuthorBible       `json:"author_bible"`           // 浣滆€呭湥缁?
	Characters       []Character       `json:"characters"`             // 瑙掕壊鍒楄〃
	Relationships    []Relationship    `json:"relationships"`          // 鍏崇郴鍒楄〃
	WorldState       []WorldStateEntry `json:"world_state"`            // 涓栫晫鐘舵€?
	OpenQuestions    []SetupQuestion   `json:"open_questions"`         // 寰呰В绛旈棶棰?
	AssistantSummary string            `json:"assistant_summary"`      // AI 鎬荤粨
	VisualDraft      *SetupVisualDraft `json:"visual_draft,omitempty"` // 鍙鍖栬崏妗?
}

// SetupRunResult 鏄缃繍琛岀粨鏋滅殑妯″瀷銆?
type SetupRunResult struct {
	RunID      string     `json:"run_id"`      // 杩愯 ID
	SessionID  string     `json:"session_id"`  // 浼氳瘽 ID
	Status     string     `json:"status"`      // 缁撴灉鐘舵€?
	SetupDraft SetupDraft `json:"setup_draft"` // 璁剧疆鑽夌
}

// ApplySetupRunResult 鏄簲鐢ㄨ缃繍琛屽悗鐨勭粨鏋溿€?
type ApplySetupRunResult struct {
	ProjectID string `json:"project_id"` // 椤圭洰 ID
	RunID     string `json:"run_id"`     // 杩愯 ID
	Status    string `json:"status"`     // 缁撴灉鐘舵€?
}

// CreateStorySessionInput 鏄垱寤烘晠浜嬩細璇濈殑杈撳叆鍙傛暟銆?
// 鏁呬簨浼氳瘽鐢ㄤ簬 AI 杈呭姪鐢熸垚鏁呬簨鍐呭銆?
type CreateStorySessionInput struct {
	Title            string // 绔犺妭/鏁呬簨鏍囬
	OpeningSituation string // 寮€灞€鎯呭
	AuthorIntent     string // 浣滆€呮剰鍥?
}

// AdvanceStorySessionInput 鏄帹杩涙晠浜嬩細璇濈殑杈撳叆鍙傛暟銆?
type AdvanceStorySessionInput struct {
	AuthorMessage string // 浣滆€呰緭鍏ョ殑娑堟伅
	BranchID      string // 缁х画鎺ㄨ繘鐨勬椂闂寸嚎鍒嗘敮 ID
	BaseTickID    string // 缁х画鎺ㄨ繘鐨勫熀纭€ tick ID
}

type ForkStoryTickInput struct {
	Name          string `json:"name"`
	AuthorMessage string `json:"author_message,omitempty"`
}

// CommitStoryRunInput 鏄彁浜ゆ晠浜嬭繍琛岀殑杈撳叆鍙傛暟銆?
type CommitStoryRunInput struct {
	DraftID       string // 鑽夌 ID
	MemoryPatchID string // 璁板繂琛ヤ竵 ID
	AuthorNote    string // 浣滆€呭娉?
}

// StorySession 鏄晠浜嬩細璇濈殑妯″瀷銆?
// 鏁呬簨浼氳瘽鏄崟涓晠浜嬬敓鎴愬懆鏈熺殑涓婁笅鏂囧鍣ㄣ€?
type StorySession struct {
	ID                         string                `json:"id"`                            // 浼氳瘽鍞竴鏍囪瘑绗?
	ProjectID                  string                `json:"project_id"`                    // 鎵€灞為」鐩?ID
	Title                      string                `json:"title"`                         // 浼氳瘽鏍囬
	OpeningSituation           string                `json:"opening_situation"`             // 寮€灞€鎯呭
	AuthorIntent               string                `json:"author_intent"`                 // 浣滆€呮剰鍥?
	LastAuthorMessage          string                `json:"last_author_message"`           // 鏈€鍚庝竴鏉′綔鑰呮秷鎭?
	Status                     string                `json:"status"`                        // 浼氳瘽鐘舵€?
	CurrentPlotVariableSummary string                `json:"current_plot_variable_summary"` // 褰撳墠鍓ф儏鍙橀噺鎽樿
	Messages                   []ConversationMessage `json:"messages"`                      // 浼氳瘽娑堟伅鍘嗗彶
	CreatedAt                  time.Time             `json:"created_at"`                    // 鍒涘缓鏃堕棿
	UpdatedAt                  time.Time             `json:"updated_at"`                    // 鏈€鍚庢洿鏂版椂闂?
}

// StoryRun 鏄晠浜嬭繍琛岀殑妯″瀷銆?
// 姣忔鐢ㄦ埛鎺ㄨ繘鏁呬簨浼氳瘽鏃跺垱寤轰竴涓繍琛岋紝璺熻釜 AI 鍐呭鐢熸垚杩涘害銆?
type StoryRun struct {
	RunID       string     `json:"run_id"`     // 杩愯鍞竴鏍囪瘑绗?
	SessionID   string     `json:"session_id"` // 鎵€灞炰細璇?ID
	ProjectID   string     `json:"project_id"` // 鎵€灞為」鐩?ID
	BranchID    string     `json:"branch_id,omitempty"`
	BaseTickID  string     `json:"base_tick_id,omitempty"`
	HeadTickID  string     `json:"head_tick_id,omitempty"`
	Status      string     `json:"status"`       // 杩愯鐘舵€?
	CurrentStep string     `json:"current_step"` // 褰撳墠姝ラ
	Progress    int        `json:"progress"`     // 杩涘害鐧惧垎姣?
	Error       string     `json:"error,omitempty"`
	CommittedAt *time.Time `json:"committed_at"` // 鎻愪氦鏃堕棿锛堝鏋滄湁锛?
	CreatedAt   time.Time  `json:"created_at"`   // 鍒涘缓鏃堕棿
	UpdatedAt   time.Time  `json:"updated_at"`   // 鏈€鍚庢洿鏂版椂闂?
}

// Draft 鏄敓鎴愮殑绔犺妭鑽夌銆?
// 鑽夌鏄€欓€夋暟鎹紝鍙湁 commit 鍚庢墠浼氬彉鎴愭寮忕珷鑺傘€?
type Draft struct {
	ID            string `json:"id"`             // 鑽夌鍞竴鏍囪瘑绗?
	Title         string `json:"title"`          // 绔犺妭鏍囬
	ChapterNumber int    `json:"chapter_number"` // 绔犺妭缂栧彿
	Content       string `json:"content"`        // 绔犺妭姝ｆ枃
	Summary       string `json:"summary"`        // 绔犺妭鎽樿
	WordCount     int    `json:"word_count"`     // 瀛楁暟缁熻
}

// PlotVariable 鏄墽鎯呭彉閲忥紝瀹氫箟鏁呬簨涓殑鏍稿績鎴忓墽鎬ч€夋嫨銆?
// AI 鍦ㄧ敓鎴愬唴瀹瑰墠浼氱‘瀹氬墽鎯呭彉閲忥紝涓鸿鑹叉彁渚涙湁鎰忎箟鐨勯€夋嫨銆?
type PlotVariable struct {
	PressureSource      string   `json:"pressure_source"`       // 鍘嬪姏鏉ユ簮
	FocalCharacterID    string   `json:"focal_character_id"`    // 鏍稿績瑙掕壊 ID
	CoreChoice          string   `json:"core_choice"`           // 鏍稿績閫夋嫨鎻忚堪
	OptionA             string   `json:"option_a"`              // 閫夐」 A
	OptionB             string   `json:"option_b"`              // 閫夐」 B
	CostA               string   `json:"cost_a"`                // 閫夋嫨 A 鐨勪唬浠?
	CostB               string   `json:"cost_b"`                // 閫夋嫨 B 鐨勪唬浠?
	IrreversibleEffect  string   `json:"irreversible_effect"`   // 涓嶅彲閫嗗奖鍝?
	RelatedCharacterIDs []string `json:"related_character_ids"` // 鐩稿叧瑙掕壊 ID 鍒楄〃
	WorldStatePressure  []string `json:"world_state_pressure"`  // 涓栫晫鐘舵€佸帇鍔?
}

type StoryTimelineEvent struct {
	ID             string   `json:"id"`
	TimeIndex      int      `json:"time_index"`
	CharacterID    string   `json:"character_id,omitempty"`
	CharacterName  string   `json:"character_name,omitempty"`
	LocationKey    string   `json:"location_key"`
	LocationName   string   `json:"location_name,omitempty"`
	ActionType     string   `json:"action_type"`
	Summary        string   `json:"summary"`
	Intent         string   `json:"intent,omitempty"`
	Visibility     string   `json:"visibility,omitempty"`
	TargetActorIDs []string `json:"target_actor_ids,omitempty"`
}

type StoryLocationGroup struct {
	ID           string   `json:"id"`
	LocationKey  string   `json:"location_key"`
	LocationName string   `json:"location_name,omitempty"`
	CharacterIDs []string `json:"character_ids"`
	EventIDs     []string `json:"event_ids"`
}

type StoryInteractionGroup struct {
	ID              string   `json:"id"`
	LocationKey     string   `json:"location_key"`
	LocationName    string   `json:"location_name,omitempty"`
	CharacterIDs    []string `json:"character_ids"`
	EventIDs        []string `json:"event_ids,omitempty"`
	ShouldInteract  bool     `json:"should_interact"`
	InteractionType string   `json:"interaction_type,omitempty"`
	Stakes          string   `json:"stakes,omitempty"`
	Rationale       string   `json:"rationale,omitempty"`
	Priority        int      `json:"priority,omitempty"`
}

type StoryInteractionAnalysis struct {
	LocationGroups    []StoryLocationGroup    `json:"location_groups"`
	InteractionGroups []StoryInteractionGroup `json:"interaction_groups"`
}

type StoryInteractionTurn struct {
	TurnIndex          int      `json:"turn_index"`
	InteractionGroupID string   `json:"interaction_group_id"`
	ActorID            string   `json:"actor_id,omitempty"`
	ActorName          string   `json:"actor_name,omitempty"`
	ActionType         string   `json:"action_type"`
	Speech             string   `json:"speech,omitempty"`
	ActionSummary      string   `json:"action_summary,omitempty"`
	TargetActorIDs     []string `json:"target_actor_ids,omitempty"`
	Intent             string   `json:"intent,omitempty"`
	LocationKey        string   `json:"location_key,omitempty"`
	LocationName       string   `json:"location_name,omitempty"`
}

type StoryInteractionTranscript struct {
	GroupID        string                 `json:"group_id"`
	LocationKey    string                 `json:"location_key"`
	LocationName   string                 `json:"location_name,omitempty"`
	CharacterIDs   []string               `json:"character_ids"`
	Turns          []StoryInteractionTurn `json:"turns"`
	OutcomeSummary string                 `json:"outcome_summary,omitempty"`
}

type StoryBranch struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"project_id"`
	SessionID        string    `json:"session_id"`
	Name             string    `json:"name"`
	BaseTickID       string    `json:"base_tick_id,omitempty"`
	HeadTickID       string    `json:"head_tick_id,omitempty"`
	Status           string    `json:"status"`
	CreatedFromRunID string    `json:"created_from_run_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type StoryTick struct {
	ID           string         `json:"id"`
	ProjectID    string         `json:"project_id"`
	SessionID    string         `json:"session_id"`
	BranchID     string         `json:"branch_id"`
	ParentTickID string         `json:"parent_tick_id,omitempty"`
	SourceRunID  string         `json:"source_run_id,omitempty"`
	Sequence     int            `json:"sequence"`
	Kind         string         `json:"kind"`
	Summary      string         `json:"summary"`
	Payload      map[string]any `json:"payload"`
	CreatedAt    time.Time      `json:"created_at"`
}

type StoryStateVersion struct {
	ID              string         `json:"id"`
	ProjectID       string         `json:"project_id"`
	EntityType      string         `json:"entity_type"`
	EntityID        string         `json:"entity_id"`
	ParentVersionID string         `json:"parent_version_id,omitempty"`
	SourceTickID    string         `json:"source_tick_id"`
	SourceRunID     string         `json:"source_run_id,omitempty"`
	Snapshot        map[string]any `json:"snapshot"`
	CreatedAt       time.Time      `json:"created_at"`
}

type StoryTickStateRef struct {
	TickID     string `json:"tick_id"`
	ProjectID  string `json:"project_id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	VersionID  string `json:"version_id"`
}

type StoryTickState struct {
	Tick     StoryTick           `json:"tick"`
	Refs     []StoryTickStateRef `json:"refs"`
	Versions []StoryStateVersion `json:"versions"`
}

type StorySessionTimeline struct {
	SessionID string        `json:"session_id"`
	Branches  []StoryBranch `json:"branches"`
	Ticks     []StoryTick   `json:"ticks"`
}

type AdvanceStoryTickInput struct {
	TickHours int `json:"tick_hours"`
}

type StoryTimeline struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	CurrentTime time.Time `json:"current_time"`
	Tick        int       `json:"tick"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StoryTickRun struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Tick        int       `json:"tick"`
	FromTime    time.Time `json:"from_time"`
	ToTime      time.Time `json:"to_time"`
	Status      string    `json:"status"`
	CurrentStep string    `json:"current_step"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WorldMap struct {
	ID         string         `json:"id"`
	ProjectID  string         `json:"project_id"`
	Name       string         `json:"name"`
	Seed       string         `json:"seed"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	Status     string         `json:"status"`
	Properties map[string]any `json:"properties,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type MapTile struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	MapID       string         `json:"map_id"`
	X           int            `json:"x"`
	Y           int            `json:"y"`
	Altitude    int            `json:"altitude"`
	Temperature int            `json:"temperature"`
	Humidity    int            `json:"humidity"`
	IsOcean     bool           `json:"is_ocean"`
	Terrain     string         `json:"terrain"`
	Properties  map[string]any `json:"properties,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type LocationState struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	MapID       string         `json:"map_id,omitempty"`
	RegionID    string         `json:"region_id,omitempty"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	X           int            `json:"x"`
	Y           int            `json:"y"`
	Radius      int            `json:"radius,omitempty"`
	Status      string         `json:"status"`
	Properties  map[string]any `json:"properties,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type FactionInfluence struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	LocationID  string    `json:"location_id"`
	FactionName string    `json:"faction_name"`
	Influence   int       `json:"influence"`
	Attitude    string    `json:"attitude"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CharacterOngoingAction struct {
	ActionType  string    `json:"action_type"`
	Description string    `json:"description"`
	StartedAt   time.Time `json:"started_at"`
	EndsAt      time.Time `json:"ends_at"`
	Status      string    `json:"status"`
	Rationale   string    `json:"rationale"`
}

type CharacterSimulationState struct {
	ID            string                  `json:"id"`
	ProjectID     string                  `json:"project_id"`
	CharacterID   string                  `json:"character_id"`
	LocationID    string                  `json:"location_id"`
	X             int                     `json:"x"`
	Y             int                     `json:"y"`
	Status        string                  `json:"status"`
	OngoingAction *CharacterOngoingAction `json:"ongoing_action,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
}

type SimulationEvent struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	TickRunID   string         `json:"tick_run_id"`
	EventName   string         `json:"event_name"`
	Sequence    int            `json:"sequence"`
	CharacterID string         `json:"character_id,omitempty"`
	LocationID  string         `json:"location_id,omitempty"`
	Summary     string         `json:"summary"`
	Payload     map[string]any `json:"payload,omitempty"`
	OccurredAt  time.Time      `json:"occurred_at"`
	CreatedAt   time.Time      `json:"created_at"`
}

type SimulationSnapshot struct {
	ID        string               `json:"id"`
	ProjectID string               `json:"project_id"`
	TickRunID string               `json:"tick_run_id"`
	Tick      int                  `json:"tick"`
	Snapshot  StorySimulationState `json:"snapshot"`
	CreatedAt time.Time            `json:"created_at"`
}

type StorySimulationState struct {
	Timeline        StoryTimeline              `json:"timeline"`
	Map             *WorldMap                  `json:"map,omitempty"`
	Tiles           []MapTile                  `json:"tiles,omitempty"`
	Locations       []LocationState            `json:"locations"`
	Factions        []FactionInfluence         `json:"factions"`
	CharacterStates []CharacterSimulationState `json:"character_states"`
	Characters      []Character                `json:"characters,omitempty"`
	LatestEvents    []SimulationEvent          `json:"latest_events,omitempty"`
}

type NearbyLocationContext struct {
	Location          LocationState      `json:"location"`
	Distance          float64            `json:"distance"`
	FactionInfluences []FactionInfluence `json:"faction_influences,omitempty"`
}

type CharacterActionDecisionInput struct {
	Timeline          StoryTimeline            `json:"timeline"`
	Character         Character                `json:"character"`
	CharacterState    CharacterSimulationState `json:"character_state"`
	Location          LocationState            `json:"location"`
	FactionInfluences []FactionInfluence       `json:"faction_influences"`
	NearbyLocations   []NearbyLocationContext  `json:"nearby_locations,omitempty"`
}

type CharacterActionDecision struct {
	ActionType    string `json:"action_type"`
	Description   string `json:"description"`
	DurationHours int    `json:"duration_hours"`
	Rationale     string `json:"rationale"`
}

type AdvanceStoryTickResult struct {
	Run      StoryTickRun         `json:"run"`
	Events   []SimulationEvent    `json:"events"`
	Snapshot SimulationSnapshot   `json:"snapshot"`
	State    StorySimulationState `json:"state"`
}

// ReviewReport 鏄闃呮姤鍛婏紝鍖呭惈 AI 瀵圭敓鎴愬唴瀹圭殑璐ㄩ噺璇勪及銆?
type ReviewReport struct {
	Pass             bool     `json:"pass"`              // 鏄惁閫氳繃
	HardViolations   []string `json:"hard_violations"`   // 纭€ц繚瑙勶紙蹇呴』淇鐨勯棶棰橈級
	ContinuityIssues []string `json:"continuity_issues"` // 杩炵画鎬ч棶棰?
	StyleIssues      []string `json:"style_issues"`      // 椋庢牸闂
	SuggestedFixes   []string `json:"suggested_fixes"`   // 寤鸿淇
}

// CharacterMemoryUpdate 鏄鑹茶蹇嗘洿鏂般€?
type CharacterMemoryUpdate struct {
	CharacterID string `json:"character_id"` // 瑙掕壊 ID
	Type        string `json:"type"`         // 鏇存柊绫诲瀷
	Content     string `json:"content"`      // 鏇存柊鍐呭
	Importance  int    `json:"importance"`   // 閲嶈鎬?
}

// RelationshipViewUpdate 鏄叧绯昏瑙掓洿鏂般€?
type RelationshipViewUpdate struct {
	ViewID                 string `json:"view_id"`                  // 瑙嗚 ID
	PairID                 string `json:"pair_id"`                  // 鍏崇郴瀵?ID
	SourceCharacterID      string `json:"source_character_id"`      // 婧愯鑹?ID
	TargetCharacterID      string `json:"target_character_id"`      // 鐩爣瑙掕壊 ID
	PublicAttitude         string `json:"public_attitude"`          // 鍏紑鎬佸害
	PrivateAttitude        string `json:"private_attitude"`         // 绉佷笅鎬佸害
	BelievedTargetAttitude string `json:"believed_target_attitude"` // 浠ヤ负瀵规柟鐨勬€佸害
	MaskingStrategy        string `json:"masking_strategy"`         // 鎺╅グ绛栫暐
}

// RelationshipUpdate 鏄叧绯绘洿鏂般€?
type RelationshipUpdate struct {
	PairID       string                   `json:"pair_id"`       // 鍏崇郴瀵?ID
	Summary      string                   `json:"summary"`       // 鏇存柊鍚庣殑鎽樿
	TensionDelta string                   `json:"tension_delta"` // 绱у紶搴﹀彉鍖?
	Pair         *RelationshipPair        `json:"pair"`          // 鏇存柊鐨勫叧绯诲锛堝彲閫夛級
	Views        []RelationshipViewUpdate `json:"views"`         // 鏇存柊鐨勮瑙掑垪琛?
	Events       []RelationshipEvent      `json:"events"`        // 鏂板鐨勪簨浠跺垪琛?
}

// WorldStateUpdate 鏄笘鐣岀姸鎬佹洿鏂般€?
type WorldStateUpdate struct {
	Key       string `json:"key"`       // 鐘舵€侀敭
	Operation string `json:"operation"` // 鎿嶄綔绫诲瀷锛坰et, update, delete锛?
	Value     any    `json:"value"`     // 鏂板€?
	Note      string `json:"note"`      // 澶囨敞璇存槑
}

// MemoryPatch 鏄蹇嗚ˉ涓侊紝灏佽浜嗘墍鏈夌姸鎬佹洿鏂般€?
// 褰撴晠浜嬭繍琛岃鎻愪氦鏃讹紝鐩稿叧鐨勭姸鎬佸彉鍖栦細閫氳繃璁板繂琛ヤ竵缁熶竴搴旂敤銆?
type MemoryPatch struct {
	ID                     string                  `json:"id"`                       // 琛ヤ竵鍞竴鏍囪瘑绗?
	Status                 string                  `json:"status"`                   // 琛ヤ竵鐘舵€?
	CharacterMemoryUpdates []CharacterMemoryUpdate `json:"character_memory_updates"` // 瑙掕壊璁板繂鏇存柊鍒楄〃
	RelationshipUpdates    []RelationshipUpdate    `json:"relationship_updates"`     // 鍏崇郴鏇存柊鍒楄〃
	WorldStateUpdates      []WorldStateUpdate      `json:"world_state_updates"`      // 涓栫晫鐘舵€佹洿鏂板垪琛?
}

// StoryRunResult 鏄晠浜嬭繍琛岀粨鏋滅殑妯″瀷銆?
type StoryRunResult struct {
	RunID                  string                       `json:"run_id"`     // 杩愯 ID
	SessionID              string                       `json:"session_id"` // 浼氳瘽 ID
	Status                 string                       `json:"status"`     // 缁撴灉鐘舵€?
	BranchID               string                       `json:"branch_id,omitempty"`
	BaseTickID             string                       `json:"base_tick_id,omitempty"`
	HeadTickID             string                       `json:"head_tick_id,omitempty"`
	PlotVariable           PlotVariable                 `json:"plot_variable"`           // 鍓ф儏鍙橀噺
	EventTimeline          []StoryTimelineEvent         `json:"event_timeline"`          // 浜嬩欢鏃堕棿绾?
	InteractionAnalysis    StoryInteractionAnalysis     `json:"interaction_analysis"`    // 浜や簰鍒嗘瀽
	InteractionTranscripts []StoryInteractionTranscript `json:"interaction_transcripts"` // 浜ゆ秹璁板綍
	Draft                  Draft                        `json:"draft"`                   // 绔犺妭鑽夌
	Review                 ReviewReport                 `json:"review"`                  // 瀹￠槄鎶ュ憡
	MemoryPatch            MemoryPatch                  `json:"memory_patch"`            // 璁板繂琛ヤ竵
}

// CommitStoryRunResult 鏄彁浜ゆ晠浜嬭繍琛屽悗鐨勭粨鏋溿€?
type CommitStoryRunResult struct {
	Chapter  Chapter     `json:"chapter"`   // 鎻愪氦鐨勭珷鑺?
	Patch    MemoryPatch `json:"patch"`     // 搴旂敤鐨勮蹇嗚ˉ涓?
	StoryRun StoryRun    `json:"story_run"` // 鍏宠仈鐨勬晠浜嬭繍琛?
}

type CreateDialogueSessionInput struct {
	Title string `json:"title"`
}

type AdvanceDialogueSessionInput struct {
	UserMessage string `json:"user_message"`
}

type ExecuteDialogueActionInput struct {
	Confirm    bool   `json:"confirm"`
	AuthorNote string `json:"author_note"`
}

type RejectDialogueActionInput struct {
	Reason string `json:"reason"`
}

type DialogueMessage struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type DialogueSession struct {
	ID              string            `json:"id"`
	ProjectID       string            `json:"project_id"`
	Title           string            `json:"title"`
	LastUserMessage string            `json:"last_user_message"`
	Status          string            `json:"status"`
	LatestRunID     string            `json:"latest_run_id,omitempty"`
	LatestRunStatus string            `json:"latest_run_status,omitempty"`
	LatestRunError  string            `json:"latest_run_error,omitempty"`
	Messages        []DialogueMessage `json:"messages"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type DialogueRun struct {
	RunID       string    `json:"run_id"`
	SessionID   string    `json:"session_id"`
	ProjectID   string    `json:"project_id"`
	Status      string    `json:"status"`
	CurrentStep string    `json:"current_step"`
	Progress    int       `json:"progress"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DialogueQuestion struct {
	Key          string `json:"key"`
	Question     string `json:"question"`
	WhyItMatters string `json:"why_it_matters"`
}

type DialogueToolTrace struct {
	ToolName  string    `json:"tool_name"`
	Summary   string    `json:"summary"`
	OK        bool      `json:"ok"`
	CreatedAt time.Time `json:"created_at"`
}

type DialogueActionOption struct {
	ID                   string         `json:"id"`
	SessionID            string         `json:"session_id"`
	RunID                string         `json:"run_id"`
	ProjectID            string         `json:"project_id"`
	ActionType           string         `json:"action_type"`
	Label                string         `json:"label"`
	Description          string         `json:"description"`
	Rationale            string         `json:"rationale"`
	ConfirmationRequired bool           `json:"confirmation_required"`
	Payload              map[string]any `json:"payload"`
	Status               string         `json:"status"`
	Result               map[string]any `json:"result,omitempty"`
	Error                string         `json:"error,omitempty"`
	ExpiresAt            *time.Time     `json:"expires_at,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

type DialogueRunResult struct {
	RunID               string                 `json:"run_id"`
	SessionID           string                 `json:"session_id"`
	Status              string                 `json:"status"`
	AssistantMessage    string                 `json:"assistant_message"`
	ActionOptions       []DialogueActionOption `json:"action_options"`
	ClarifyingQuestions []DialogueQuestion     `json:"clarifying_questions"`
	SuggestedReplies    []string               `json:"suggested_replies"`
	ContextSummary      string                 `json:"context_summary"`
	ToolTrace           []DialogueToolTrace    `json:"tool_trace"`
}

// Chapter 鏄凡鎻愪氦鐨勬晠浜嬬珷鑺傘€?
// 绔犺妭鏄晠浜嬬殑鍩烘湰鍗曚綅锛屾瘡涓彁浜ょ殑鏁呬簨杩愯浼氫骇鐢熶竴涓柊鐨勭珷鑺傘€?
type Chapter struct {
	ID            string    `json:"id"`             // 绔犺妭鍞竴鏍囪瘑绗?
	ProjectID     string    `json:"project_id"`     // 鎵€灞為」鐩?ID
	ChapterNumber int       `json:"chapter_number"` // 绔犺妭缂栧彿锛堥€掑锛?
	Title         string    `json:"title"`          // 绔犺妭鏍囬
	Summary       string    `json:"summary"`        // 绔犺妭鎽樿
	Content       string    `json:"content"`        // 绔犺妭姝ｆ枃
	AuthorNote    string    `json:"author_note"`    // 浣滆€呭娉?
	Status        string    `json:"status"`         // 绔犺妭鐘舵€?
	WordCount     int       `json:"word_count"`     // 瀛楁暟缁熻
	CommittedAt   time.Time `json:"committed_at"`   // 鎻愪氦鏃堕棿
}

// CreateMemoryInput 鏄垱寤鸿鑹茶蹇嗙殑杈撳叆鍙傛暟銆?
type CreateMemoryInput struct {
	Content    string // 璁板繂鍐呭
	Importance int    // 閲嶈鎬х瓑绾?
	Note       string // 澶囨敞璇存槑
}

// Memory 鏄鑹茬殑璁板繂鏉＄洰銆?
// 璁板繂鏄鑹插湪鏁呬簨涓粡鍘嗕簨浠剁殑璁板綍锛岀敤浜?AI 鐢熸垚鏃剁殑涓婁笅鏂囧弬鑰冦€?
type Memory struct {
	ID              string    `json:"id"`                // 璁板繂鍞竴鏍囪瘑绗?
	CharacterID     string    `json:"character_id"`      // 鎵€灞炶鑹?ID
	Content         string    `json:"content"`           // 璁板繂鍐呭
	SourceChapterID string    `json:"source_chapter_id"` // 鏉ユ簮绔犺妭 ID
	Importance      int       `json:"importance"`        // 閲嶈鎬х瓑绾?
	Note            string    `json:"note"`              // 澶囨敞璇存槑
	Status          string    `json:"status"`            // 鐘舵€佹爣璇?
	CreatedAt       time.Time `json:"created_at"`        // 鍒涘缓鏃堕棿
}

// RunEvent 鏄繍琛屼簨浠讹紝鐢ㄤ簬璁板綍 AI 鐢熸垚杩囩▼涓殑鍏抽敭浜嬩欢銆?
// 浜嬩欢閫氳繃 SSE 瀹炴椂鎺ㄩ€佺粰瀹㈡埛绔紝骞舵寔涔呭寲鐢ㄤ簬瀹¤銆?

// RunExecutionWork 是可恢复运行执行器从持久存储扫描到的待处理工作项。
// 该类型属于 run execution 边界；当前保留在 types.go，后续模型拆分时可迁移到执行模型文件。
type RunExecutionWork struct {
	RunKind   string    `json:"run_kind"`
	RunID     string    `json:"run_id"`
	SessionID string    `json:"session_id"`
	ProjectID string    `json:"project_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RunEvent struct {
	ID        string         `json:"id"`         // 浜嬩欢鍞竴鏍囪瘑绗?
	RunKind   string         `json:"run_kind"`   // 杩愯绫诲瀷锛坰etup/story锛?
	RunID     string         `json:"run_id"`     // 鎵€灞炶繍琛?ID
	EventName string         `json:"event_name"` // 浜嬩欢鍚嶇О
	Sequence  int            `json:"sequence"`   // 浜嬩欢搴忓彿
	Payload   map[string]any `json:"payload"`    // 浜嬩欢闄勫姞鏁版嵁
	CreatedAt time.Time      `json:"created_at"` // 浜嬩欢鍙戠敓鏃堕棿
}

// StateRevision 鏄姸鎬佷慨璁㈠揩鐓с€?
// 鍦ㄩ噸瑕佺姸鎬佸彉鏇村墠淇濆瓨蹇収锛岀敤浜庡洖婊氭垨瀹¤銆?
type StateRevision struct {
	ID          string         // 淇鍞竴鏍囪瘑绗?
	ProjectID   string         // 鎵€灞為」鐩?ID
	EntityType  string         // 瀹炰綋绫诲瀷锛堝 character, relationship锛?
	EntityID    string         // 瀹炰綋 ID
	SourceRunID string         // 鏉ユ簮杩愯 ID
	Snapshot    map[string]any // 鐘舵€佸揩鐓ф暟鎹?
	CreatedAt   time.Time      // 淇鏃堕棿
}
