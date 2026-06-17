import type { ScoreLevel } from "./types";

const levelStyles: Record<ScoreLevel, string> = {
    EE: "bg-teal-100 text-teal-800",
    ME: "bg-blue-100 text-blue-800",
    AE: "bg-amber-100 text-amber-800",
    BE: "bg-red-100 text-red-800",
};

export function ScoreLevelBadge({ level }: { level: ScoreLevel }) {
    return (
        <span
            className={`inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-semibold ${levelStyles[level]}`}
        >
            {level}
        </span>
    );
}
