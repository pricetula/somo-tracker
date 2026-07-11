/**
 * StreamPill — renders a stream name as a badge with a coloured dot indicator.
 *
 * The dot colour comes from the stream's `color` field (a hex value).
 *
 * Usage in a DataTable column:
 * ```tsx
 * import { StreamPill } from "@/features/settings-school";
 *
 * {
 *   id: "stream_name",
 *   header: "Stream",
 *   cell: (row) => <StreamPill name={row.stream_name} color={row.stream_color} />,
 * }
 * ```
 */

import { Badge } from "@/components/ui/badge";

// ─── Props ─────────────────────────────────────────────────────────────────

interface StreamPillProps {
    /** The stream display name. */
    name: string;
    /** The stream's colour hex value (e.g. "#ef4444", "#3b82f6"). */
    color?: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function StreamPill({ name, color }: StreamPillProps) {
    if (!color) {
        return (
            <Badge variant="secondary" className="font-normal">
                {name}
            </Badge>
        );
    }

    return (
        <Badge variant="ghost" className="gap-1.5 p-0">
            <span className="inline-block size-1 rounded-full" style={{ backgroundColor: color }} />
            {name}
        </Badge>
    );
}
