/**
 * Dashboard feature — public API barrel.
 *
 * Import only from this barrel, never from internal paths.
 */

// Components
export { QuickStats } from "./components/quick-stats";
export type { StatItem } from "./components/quick-stats";
export { PendingItems } from "./components/pending-items";
export type { PendingItemData } from "./components/pending-items";
export { ActivityFeed } from "./components/activity-feed";

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
