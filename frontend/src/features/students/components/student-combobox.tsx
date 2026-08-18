/**
 * StudentCombobox — reusable student selector built on Radix-only Combobox.
 *
 * Fetches its own options internally — zero prop drilling.
 * Stable TanStack Query key → 1 network call for N instances.
 */

"use client";

import * as React from "react";
import Link from "next/link";

import { cn } from "@/lib/utils";
import { Combobox, ComboboxChip } from "@/components/ui/combobox";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { getErrorMessage } from "@/lib/errors";
import { useStudents } from "@/features/students/hooks/use-students";

// ─── Props ────────────────────────────────────────────────────────────────

export interface StudentComboboxProps {
    /** Currently selected student ID(s) (controlled). */
    value: string | string[];
    /** Called when a student is selected / deselected. */
    onChange: (value: string | string[]) => void;
    /** Placeholder text when nothing is selected. */
    placeholder?: string;
    /** Optional outer container class. */
    className?: string;
    /** Allow selecting multiple students (default: false). */
    isMultiSelect?: boolean;
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

// ─── Component ────────────────────────────────────────────────────────────

export function StudentCombobox({
    value,
    onChange,
    placeholder = "Select a student...",
    className,
    isMultiSelect = false,
    onCreateItem,
    doPreselectFirstOption = false,
}: StudentComboboxProps) {
    const { data, isLoading, isError, error } = useStudents({ limit: 500 });

    const items = React.useMemo(
        () =>
            data?.items.map((s) => ({
                value: s.id,
                label: s.full_name,
            })) ?? [],
        [data]
    );

    // ── Auto-preselect first option ──────────────────────────────────────
    const hasPreselected = React.useRef(false);
    React.useEffect(() => {
        if (!doPreselectFirstOption || items.length === 0 || hasPreselected.current) return;

        const hasValue = isMultiSelect ? (value as string[]).length > 0 : (value as string) !== "";

        if (hasValue) {
            hasPreselected.current = true;
            return;
        }

        hasPreselected.current = true;
        onChange(isMultiSelect ? [items[0].value] : items[0].value);
    }, [doPreselectFirstOption, items, isMultiSelect, value, onChange]);

    // ── Loading state ─────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className={cn("w-full", className)}>
                <Skeleton className="h-9 w-full" />
            </div>
        );
    }

    // ── Error state ──────────────────────────────────────────────────────
    if (isError) {
        return (
            <div className={cn("w-full", className)}>
                <Alert variant="destructive" className="h-9 items-center py-0 text-xs">
                    <AlertDescription>{getErrorMessage(error)}</AlertDescription>
                </Alert>
            </div>
        );
    }

    // ── No items at all ──────────────────────────────────────────────────
    if (items.length === 0) {
        return (
            <Alert className="text-muted-foreground h-9 items-center py-0 text-xs">
                <AlertDescription>
                    No students found.{" "}
                    <Link href="/students/new" className="underline underline-offset-2">
                        Add one
                    </Link>
                    .
                </AlertDescription>
            </Alert>
        );
    }

    // ── Single-select ───────────────────────────────────────────────────
    if (!isMultiSelect) {
        return (
            <Combobox
                items={items}
                value={value as string}
                onValueChange={onChange}
                placeholder={placeholder}
                emptyText="No student found."
                className={cn("w-full", className)}
                onCreateItem={onCreateItem}
            />
        );
    }

    // ── Multi-select ────────────────────────────────────────────────────
    return (
        <Combobox
            items={items}
            value={value as string[]}
            onValueChange={onChange}
            multiple
            placeholder={placeholder}
            emptyText="No student found."
            className={cn("w-full", className)}
            onCreateItem={onCreateItem}
            renderTrigger={({ selectedItems }) =>
                selectedItems.length > 0 ? (
                    <span className="flex flex-wrap gap-1">
                        {selectedItems.map((item) => (
                            <ComboboxChip
                                key={item.value}
                                value={item.value}
                                onRemove={(v) => {
                                    const next = (value as string[]).filter((id) => id !== v);
                                    onChange(next);
                                }}
                            >
                                {item.label}
                            </ComboboxChip>
                        ))}
                    </span>
                ) : (
                    <span className="text-muted-foreground truncate">{placeholder}</span>
                )
            }
        />
    );
}
