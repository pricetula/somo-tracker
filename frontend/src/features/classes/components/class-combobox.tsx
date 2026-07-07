/**
 * ClassCombobox — reusable class selector built on Radix-only Combobox.
 *
 * Fetches its own options internally — zero prop drilling.
 * Stable TanStack Query key → 1 network call for N instances.
 * No React Context — each instance is fully isolated.
 */

"use client";

import * as React from "react";

import { cn } from "@/lib/utils";
import { Combobox, ComboboxChip } from "@/components/ui/combobox";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { getErrorMessage } from "@/lib/errors";

import { useClassList } from "../hooks/use-classes";

// ─── Props ────────────────────────────────────────────────────────────────

export interface ClassComboboxProps {
    /** Currently selected class ID(s) (controlled). */
    value: string | string[];
    /** Called when a class is selected / deselected. */
    onChange: (value: string | string[]) => void;
    /** Placeholder text when nothing is selected. */
    placeholder?: string;
    /** Optional outer container class. */
    className?: string;
    /** Allow selecting multiple classes (default: false). */
    isMultiSelect?: boolean;
}

// ─── Component ────────────────────────────────────────────────────────────

export function ClassCombobox({
    value,
    onChange,
    placeholder = "Select a class...",
    className,
    isMultiSelect = false,
}: ClassComboboxProps) {
    const { data, isLoading, isError, error } = useClassList();

    const items = data?.items ?? [];

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

    // ── Single-select ───────────────────────────────────────────────────
    if (!isMultiSelect) {
        return (
            <Combobox
                items={items}
                value={value as string}
                onValueChange={onChange}
                placeholder={placeholder}
                emptyText="No class found."
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
            emptyText="No class found."
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
