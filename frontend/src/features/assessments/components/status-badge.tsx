/**
 * StatusBadge — Displays an assessment session status as a coloured pill.
 */

import { Badge } from "@/components/ui/badge";
import { SESSION_STATUS_LABELS, SESSION_STATUS_COLORS } from "../types";

interface Props {
    status: string | null | undefined;
}

export function StatusBadge({ status }: Props) {
    if (!status) return null;

    return (
        <Badge
            variant="secondary"
            className={`${SESSION_STATUS_COLORS[status] ?? ""} text-xs font-medium`}
        >
            {SESSION_STATUS_LABELS[status] ?? status}
        </Badge>
    );
}
