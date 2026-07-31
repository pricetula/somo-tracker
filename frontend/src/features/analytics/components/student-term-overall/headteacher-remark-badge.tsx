/**
 * HeadteacherRemarkBadge — Display a headteacher's remark as a styled callout.
 *
 * Visualisation: A callout displaying the headteacher's remark for the term.
 * Props: { remark, termName }.
 */
"use client";

// ─── Component ────────────────────────────────────────────────────────────

interface HeadteacherRemarkBadgeProps {
    remark: string;
    termName?: string;
}

export function HeadteacherRemarkBadge({ remark, termName }: HeadteacherRemarkBadgeProps) {
    if (!remark) {
        return null;
    }

    return (
        <div className="bg-muted/30 space-y-2 rounded-md p-4">
            <div className="flex items-center gap-2">
                <span className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                    Headteacher&apos;s Remark
                </span>
                {termName && <span className="text-muted-foreground/60 text-xs">— {termName}</span>}
            </div>
            <blockquote className="text-foreground border-primary border-l-2 pl-3 text-sm leading-relaxed italic">
                &ldquo;{remark}&rdquo;
            </blockquote>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────
