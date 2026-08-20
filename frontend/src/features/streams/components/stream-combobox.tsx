/**
 * StreamCombobox — reusable stream selector built on the shared Combobox.
 *
 * Fetches its own options via useStreamList — zero prop drilling.
 * Place in its own streams feature so all consumers import from one place.
 */

"use client";

import * as React from "react";
import Link from "next/link";
import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";
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
    /**
     * When true, automatically selects the first option if no value is set.
     * Defaults to false.
     */
    doPreselectFirstOption?: boolean;
}

// ─── Constants ────────────────────────────────────────────────────────────

const CREATE_ITEM_VALUE = "__create__";

// ─── Component ────────────────────────────────────────────────────────────

export function StreamCombobox({
    value,
    onChange,
    placeholder = "Select a stream...",
    className,
    onCreateItem,
    doPreselectFirstOption = false,
}: StreamComboboxProps) {
    const { data, isLoading, isError, error } = useStreamList();

    const items = data?.items ?? [];

    // ── Auto-preselect first option ──────────────────────────────────────
    const hasPreselected = React.useRef(false);
    React.useEffect(() => {
        const list = data?.items;
        if (!doPreselectFirstOption || !list || list.length === 0 || hasPreselected.current) return;
        if (value) {
            hasPreselected.current = true;
            return;
        }
        hasPreselected.current = true;
        onChange(list[0].id);
    }, [doPreselectFirstOption, data, value, onChange]);

    // ── Handle create item selection ──────────────────────────────────────
    const handleValueChange = React.useCallback(
        (newValue: string | null) => {
            if (newValue === CREATE_ITEM_VALUE && onCreateItem) {
                onCreateItem("");
            } else {
                onChange(newValue as string);
            }
        },
        [onChange, onCreateItem]
    );

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
                    <Link href="/streams/add" className="underline underline-offset-2">
                        Add one
                    </Link>
                    .
                </AlertDescription>
            </Alert>
        );
    }

    const comboboxItems = items.map((s) => ({ value: s.id, label: s.name }));

    return React.createElement(
        Combobox,
        { value, onValueChange: handleValueChange, className },
        React.createElement(ComboboxInput, { placeholder }),
        React.createElement(
            ComboboxContent,
            null,
            React.createElement(
                ComboboxList,
                null,
                comboboxItems.map((item) =>
                    React.createElement(
                        ComboboxItem,
                        { key: item.value, value: item.value },
                        item.label
                    )
                ),
                onCreateItem &&
                    React.createElement(
                        ComboboxItem,
                        {
                            value: CREATE_ITEM_VALUE,
                            className: "bg-muted/50 text-muted-foreground italic",
                        },
                        "Create new stream..."
                    ),
                React.createElement(ComboboxEmpty, null, "No streams found")
            )
        )
    );
}
