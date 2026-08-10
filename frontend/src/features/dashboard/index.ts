/**
 * Dashboard feature — public API barrel.
 *
 * Import only from this barrel, never from internal paths.
 */

// Components
export { QuickActions } from "./components/quick-actions";
export type { QuickAction } from "./components/quick-actions";
export { QuickStats } from "./components/quick-stats";
export type { StatItem } from "./components/quick-stats";
export { PendingItems } from "./components/pending-items";
export type { PendingItemData } from "./components/pending-items";
export { ActivityFeed } from "./components/activity-feed";

// Action configurations
export {
    TEACHER_ACTIONS,
    SCHOOL_ADMIN_ACTIONS,
    PARENT_ACTIONS,
    NURSE_ACTIONS,
    FINANCE_ACTIONS,
    SYSTEM_ADMIN_ACTIONS,
} from "./actions";

// Hooks
export {
    useDashboardCounts,
    useDashboardPendingItems,
    useDashboardSetupProgress,
    useDashboardRecentActivity,
} from "./hooks/use-dashboard-summary";

// Types
export type {
    DashboardCounts,
    PendingItem,
    SetupChecklistItem,
    ActivityItem,
} from "./hooks/use-dashboard-summary";
