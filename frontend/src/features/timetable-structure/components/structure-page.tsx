/**
 * StructurePage — main container for the Timetable Settings portal.
 *
 * Two states:
 *   1. No blocks → "Add Blueprint" button opens a dialog to define the
 *      full day's block sequence (saved for all 5 weekdays).
 *   2. Blocks exist → slot grid for assigning classes to periods.
 */

"use client";

// ─── Component ─────────────────────────────────────────────────────────────

export function StructurePage() {
    return <article>Time table</article>;
}
