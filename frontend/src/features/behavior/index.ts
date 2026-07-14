/**
 * Behavior feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

// Components
export { BehaviorReviewQueue } from "./components/behavior-review-queue";
export { CreateBehaviorNoteDialog } from "./components/create-behavior-note-dialog";
export { BehaviorCategoryManager } from "./components/behavior-category-manager";

// Hooks
export {
    useBehaviorCategories,
    useCreateBehaviorCategory,
    useUpdateBehaviorCategory,
    useCreateBehaviorNote,
    useBehaviorPendingQueue,
    useReviewBehaviorNote,
    behaviorKeys,
} from "./hooks/use-behavior";

// Types
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
} from "./types";
