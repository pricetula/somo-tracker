/**
 * Query configuration constants for TanStack Query.
 *
 * Defines stale time tiers used across all feature hooks.
 * Using named constants instead of magic numbers makes intent clear
 * and enables bulk adjustments.
 *
 * Tiers:
 *   REFERENCE_DATA — data that rarely changes (streams, terms, fee-categories)
 *   STANDARD       — default for most lists (60s)
 *   FREQUENT       — data updated in real-time (attendance rosters)
 *   LIVE           — polling data (import progress, behavior queue)
 */

export const STALE_TIMES = {
    /** Reference/lookup data — cache for 5 minutes. */
    REFERENCE_DATA: 5 * 60 * 1000,
    /** Default stale time for standard lists and details. */
    STANDARD: 60_000,
    /** Frequently updated data — recheck every 30 seconds. */
    FREQUENT: 30_000,
    /** Live/polling data — recheck every 10-15 seconds. */
    LIVE: 15_000,
} as const;
