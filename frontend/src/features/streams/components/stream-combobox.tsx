/**
 * StreamCombobox — reusable stream selector built on the shared Combobox.
 *
 * Fetches its own options via useStreamList — zero prop drilling.
 * Place in its own streams feature so all consumers import from one place.
 */

"use client";

import Link from "next/link";
import { Combobox } from "@/components/ui/combobox";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { getErrorMessage } from "@/lib/errors";
import { useStreamList } from "../hooks/use-streams";

// ─── Props ────────────────────────────────────────────────────────────────

export interface StreamComboboxProps {
    /** Currently selected stream ID (controlled). */
    value: string;
    /** Called when a stream is selected. */
    onChange: (value: string) => void;
    /** Placeholder text when nothing is selected. */
    placeholder?: string;
    /** Optional outer container class. */
    className?: string;
    /**
     * When search yields no results, shows a "Create" option.
     * If omitted, no create option is shown.
     */
    onCreateItem?: (search: string) => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function StreamCombobox({
    value,
    onChange,
    placeholder = "Select a stream...",
    className,
    onCreateItem,
}: StreamComboboxProps) {
    const { data, isLoading, isError, error } = useStreamList();

    const items = data?.items ?? [];

    // ── Loading state ─────────────────────────────────────────────────────
    if (isLoading) {
        return <Skeleton className={className ?? "h-9 w-full"} />;
    }

    // ── Error state ──────────────────────────────────────────────────────
    if (isError) {
        return (
            <Alert variant="destructive" className="h-9 items-center py-0 text-xs">
                <AlertDescription>{getErrorMessage(error)}</AlertDescription>
            </Alert>
        );
    }

    // ── No items at all (not just search miss) ───────────────────────────
    if (items.length === 0) {
        return (
            <Alert className="text-muted-foreground h-9 items-center py-0 text-xs">
                <AlertDescription>
                    No streams configured.{" "}
                    <Link href="/streams/add" className="text-primary underline underline-offset-2">
                        Add one
                    </Link>
                    .
                </AlertDescription>
            </Alert>
        );
    }

    return (
        <Combobox
            items={items.map((s) => ({ value: s.id, label: s.name }))}
            value={value}
            onValueChange={(v) => onChange(v as string)}
            placeholder={placeholder}
            className={className}
            onCreateItem={onCreateItem}
        />
    );
}
