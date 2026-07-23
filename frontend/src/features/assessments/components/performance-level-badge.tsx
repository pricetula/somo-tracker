/**
 * PerformanceLevelBadge — Displays a CBC rubric level (EE, ME, AE, BE) as a
 * coloured pill.
 */

import { Badge } from "@/components/ui/badge";
import { PERFORMANCE_LEVEL_LABELS } from "../types";

const levelColors: Record<string, string> = {
    EE: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
    ME: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
    AE: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    BE: "bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-400",
};

interface Props {
    level: string | null | undefined;
    /** When true, shows the full label (e.g. "EE — Exceeding Expectation"). */
    showLabel?: boolean;
}

export function PerformanceLevelBadge({ level, showLabel }: Props) {
    if (!level) return null;

    return (
        <Badge variant="secondary" className={`${levelColors[level] ?? ""} text-xs font-medium`}>
            {level}
            {showLabel && PERFORMANCE_LEVEL_LABELS[level] && (
                <span className="ml-1.5 font-normal">{PERFORMANCE_LEVEL_LABELS[level]}</span>
            )}
        </Badge>
    );
}
