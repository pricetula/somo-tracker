/**
 * TeacherCombobox — reusable teacher selector built on Radix-only Combobox.
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
import { useTeachers } from "../hooks/use-teachers";

// ─── Props ────────────────────────────────────────────────────────────────

export interface TeacherComboboxProps {
    /** Currently selected teacher ID(s) (controlled). */
    value: string | string[];
    /** Called when a teacher is selected / deselected. */
    onChange: (value: string | string[]) => void;
    /** Placeholder text when nothing is selected. */
    placeholder?: string;
    /** Optional outer container class. */
    className?: string;
    /** Allow selecting multiple teachers (default: false). */
    isMultiSelect?: boolean;
    /**
     * When true, automatically selects the first option if no value is set.
     * Defaults to false.
     */
    doPreselectFirstOption?: boolean;
}

// ─── Component ────────────────────────────────────────────────────────────

export function TeacherCombobox({
    value,
    onChange,
    placeholder = "Select a teacher...",
    className,
    isMultiSelect = false,
    doPreselectFirstOption = false,
}: TeacherComboboxProps) {
    const { data, isLoading, isError, error } = useTeachers({ limit: 500 });

    const items = React.useMemo(
        () =>
            data?.items.map((t) => ({
                value: t.id,
                label: t.full_name,
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
                    No teachers found.{" "}
                    <Link href="/teachers/import" className="underline underline-offset-2">
                        Invite teachers
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
                emptyText="No teacher found."
                className={cn("w-full", className)}
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
            emptyText="No teacher found."
            className={cn("w-full", className)}
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
