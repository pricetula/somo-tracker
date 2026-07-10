/**
 * GradeLevelPill — renders a grade level as a coloured pill with a
 * leading dot indicator.
 *
 * Colours are grouped by CBC education level stage so related grades
 * share the same visual hue for quick scanning.
 *
 * Usage in a DataTable column:
 * ```tsx
 * import { GradeLevelPill } from "@/features/grade-level";
 *
 * {
 *   id: "grade_level",
 *   header: "Grade",
 *   cell: (row) => <GradeLevelPill grade={row.grade_level} />,
 * }
 * ```
 */

import { Badge } from "@/components/ui/badge";
import { GRADE_LEVEL_STYLES, GRADE_LEVEL_LABELS } from "../types";

// ─── Props ─────────────────────────────────────────────────────────────────

interface GradeLevelPillProps {
    /** The grade level value to render (e.g. "G1", "PP1"). */
    grade: string;
    /** Optional override for the display label. Falls back to GRADE_LEVEL_LABELS. */
    label?: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function GradeLevelPill({ grade, label }: GradeLevelPillProps) {
    const styles = GRADE_LEVEL_STYLES[grade];
    const displayLabel = label ?? GRADE_LEVEL_LABELS[grade] ?? grade;

    // Unknown grade — render a muted fallback with no dot
    if (!styles) {
        return (
            <Badge variant="secondary" className="font-normal">
                {displayLabel}
            </Badge>
        );
    }

    return (
        <Badge variant="ghost" className="gap-1.5 p-0">
            <span className={["inline-block size-1 rounded-full", styles.dot].join(" ")} />
            {displayLabel}
        </Badge>
    );
}
