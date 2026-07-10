/**
 * EducationLevelPill — renders an education level as a coloured pill with
 * a leading dot indicator.
 *
 * Usage in a DataTable column:
 * ```tsx
 * import { EducationLevelPill } from "@/features/education-level";
 *
 * {
 *   id: "education_level",
 *   header: "Education Level",
 *   cell: (row) => <EducationLevelPill level={row.education_level} />,
 * }
 * ```
 */

import { Badge } from "@/components/ui/badge";
import { EDUCATION_LEVEL_STYLES, EDUCATION_LEVEL_LABELS } from "../types";
import type { EducationLevel } from "../types";

// ─── Props ─────────────────────────────────────────────────────────────────

interface EducationLevelPillProps {
    /** The education level value to render. */
    level: string;
    /** Optional override for the display label. Falls back to EDUCATION_LEVEL_LABELS. */
    label?: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function EducationLevelPill({ level, label }: EducationLevelPillProps) {
    const styles = EDUCATION_LEVEL_STYLES[level as EducationLevel];
    const displayLabel = label ?? EDUCATION_LEVEL_LABELS[level as EducationLevel] ?? level;

    // Unknown level — render a muted fallback with no dot
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
