package domain

const (
	SessionStatusIdle                 = "idle"
	SessionStatusAdvancing            = "advancing"
	SessionStatusReviewing            = "reviewing"
	SessionStatusAwaitingConfirmation = "awaiting_confirmation"
	SessionStatusCommitted            = "committed"
	SessionStatusFailed               = "failed"
)

const (
	RunStatusQueued                  = "queued"
	RunStatusLoadingState            = "loading_state"
	RunStatusPlanningActions         = "planning_actions"
	RunStatusExecutingAction         = "executing_action"
	RunStatusSelectingConflictAxis   = "selecting_conflict_axis"
	RunStatusGeneratingPlotVariable  = "generating_plot_variable"
	RunStatusPlanningEvents          = "planning_events"
	RunStatusSelectingInteractions   = "selecting_interactions"
	RunStatusNegotiatingInteractions = "negotiating_interactions"
	RunStatusDrivingCharacterTurns   = "driving_character_turns"
	RunStatusWritingNarrative        = "writing_narrative"
	RunStatusCheckingContinuity      = "checking_continuity"
	RunStatusGeneratingMemoryPatch   = "generating_memory_patch"
	RunStatusCompleted               = "completed"
	RunStatusCut                     = "cut"
	RunStatusFailed                  = "failed"
	RunStatusCancelled               = "cancelled"
)

const (
	EventGenerationStep            = "generation_step"
	EventStoryOrchestrationStarted = "story_orchestration_started"
	EventPlotVariable              = "plot_variable"
	EventStoryEventPlanned         = "story_event_planned"
	EventSameLocationCandidates    = "same_location_candidates"
	EventInteractionAnalysis       = "interaction_analysis"
	EventInteractionSelected       = "interaction_selected"
	EventNegotiationTurn           = "negotiation_turn"
	EventCharacterTurn             = "character_turn"
	EventDraftDelta                = "draft_delta"
)

const (
	CharacterActionStatusOngoing   = "ongoing"
	CharacterActionStatusCompleted = "completed"
)

const (
	DialogueActionStatusPending   = "pending"
	DialogueActionStatusConfirmed = "confirmed"
	DialogueActionStatusExecuting = "executing"
	DialogueActionStatusExecuted  = "executed"
	DialogueActionStatusRejected  = "rejected"
	DialogueActionStatusFailed    = "failed"
)

const (
	DialogueActionSetupStartAndAdvance  = "setup_start_and_advance"
	DialogueActionSetupAdvance          = "setup_advance"
	DialogueActionSetupApply            = "setup_apply"
	DialogueActionStoryCreateAndAdvance = "story_create_and_advance"
	DialogueActionStoryAdvance          = "story_advance"
	DialogueActionStoryCutChapter       = "story_cut_chapter"
	DialogueActionStoryForkFromEvent    = "story_fork_from_event"
	DialogueActionStoryRollbackToEvent  = "story_rollback_to_event"
)
