/**
 * Route parsing and label-mapping utilities for the Smart Breadcrumb component.
 *
 * This module is independently testable — no React or Next.js dependencies.
 */

// ─── Types ──────────────────────────────────────────────────────────────────

export type BreadcrumbSegment = {
    /** Human-readable display label (may be truncated UUID or mapped name). */
    label: string;
    /** Absolute path up to and including this segment. */
    href: string;
    /** True only for the final segment (rendered without a link). */
    isLast: boolean;
};

// ─── Static route map ───────────────────────────────────────────────────────

/**
 * Lookup table that maps known URL path segments to human-readable labels.
 * Add a single line here when a new route is introduced.
 */
const STATIC_ROUTE_LABELS: Record<string, string> = {
    admins: "Admins",
    finance: "Finance",
    nurses: "Nurses",
    tests: "Tests",
    questions: "Questions",
    add: "Add",
    // extend here as new routes are added
};

// ─── Helpers ────────────────────────────────────────────────────────────────

/**
 * Returns the first `length` characters of a segment followed by "…".
 * Default length is 6.
 */
export function truncateId(segment: string, length = 6): string {
    return segment.slice(0, length) + "…";
}

/**
 * Determines whether a path segment is a dynamic ID (UUID, long hash, etc.).
 *
 * A segment is considered a dynamic ID if it:
 * 1. Contains a hyphen (`-`), OR
 * 2. Is longer than 12 characters, OR
 * 3. Matches a UUID-like pattern (`/^[0-9a-f-]{8,}$/i`)
 */
function isDynamicId(segment: string): boolean {
    if (segment.includes("-")) return true;
    if (segment.length > 12) return true;
    if (/^[0-9a-f-]{8,}$/i.test(segment)) return true;
    return false;
}

/**
 * Capitalises the first character of a string.
 */
function capitalise(str: string): string {
    if (str.length === 0) return str;
    return str.charAt(0).toUpperCase() + str.slice(1);
}

// ─── Public API ─────────────────────────────────────────────────────────────

/**
 * Resolves a single URL path segment to a human-readable label.
 *
 * Resolution order (checked in order):
 * 1. **Static route map** — exact match in `STATIC_ROUTE_LABELS`.
 * 2. **Dynamic ID** — truncated with `truncateId` (first 6 chars + "…").
 * 3. **Fallback** — first letter capitalised.
 */
export function resolveSegmentLabel(segment: string): string {
    const lower = segment.toLowerCase();

    // Case 1: Static route map lookup
    if (lower in STATIC_ROUTE_LABELS) {
        return STATIC_ROUTE_LABELS[lower];
    }

    // Case 2: UUID / long ID detection
    if (isDynamicId(segment)) {
        return truncateId(segment);
    }

    // Case 3: Fallback — capitalise first letter
    return capitalise(segment);
}

/**
 * Parses a raw Next.js `pathname` string into an ordered array of
 * `BreadcrumbSegment` objects.
 *
 * - Always prepends a `{ label: 'Dashboard', href: '/' }` segment.
 * - Splits the pathname on `/`, filters empties, and builds cumulative hrefs.
 * - Each raw segment is resolved via `resolveSegmentLabel`.
 * - Only the final segment has `isLast: true`.
 */
export function buildBreadcrumbs(pathname: string): BreadcrumbSegment[] {
    const parts = pathname.split("/").filter(Boolean);

    const segments: BreadcrumbSegment[] = [{ label: "Dashboard", href: "/", isLast: false }];

    let cumulativeHref = "";

    for (const part of parts) {
        cumulativeHref += "/" + part;
        segments.push({
            label: resolveSegmentLabel(part),
            href: cumulativeHref,
            isLast: false,
        });
    }

    // Mark the last segment (skip if only Dashboard)
    if (segments.length > 1) {
        segments[segments.length - 1].isLast = true;
    }

    return segments;
}
