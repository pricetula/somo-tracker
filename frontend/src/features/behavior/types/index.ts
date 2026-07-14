/**
 * Behavior feature types.
 *
 * These mirror the backend API response shapes. The canonical definitions
 * live in src/lib/api/behavior.ts; this barrel re-exports them so feature
 * consumers can import from @/features/behavior/types.
 */

export type {
    BehaviorNoteStatus,
    BehaviorSeverity,
    BehaviorCategory,
    CreateCategoryPayload,
    UpdateCategoryPayload,
    BehaviorNote,
    CreateNotePayload,
    PendingNoteItem,
    PendingNotesResponse,
    ReviewDecisionPayload,
} from "@/lib/api/behavior";
