/**
 * ClassCombobox — reusable, performance-optimised class selector.
 *
 * Designed to be mounted 20–40× on a single page without input lag:
 *  • Fetches its own options internally — zero prop drilling
 *  • Stable TanStack Query key → 1 network call for N instances
 *  • Filtering handled natively by the Base UI Combobox
 *  • No React Context — each instance is fully isolated
 *
 * Follows the official shadcn combobox pattern:
 *   https://ui.shadcn.com/docs/components/base/combobox
 */

"use client";

import * as React from "react";

import { cn } from "@/lib/utils";
import {
    Combobox,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxInput,
    ComboboxItem,
    ComboboxList,
    ComboboxChip,
    ComboboxChips,
    ComboboxChipsInput,
    ComboboxValue,
} from "@/components/ui/combobox";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { getErrorMessage } from "@/lib/errors";

import { useClassList } from "../hooks/use-classes";
import type { ClassOption } from "../types";

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

    // ── Error state (required by frontend AGENTS.md §9) ──────────────────
    if (isError) {
        return (
            <div className={cn("w-full", className)}>
                <Alert variant="destructive" className="h-9 items-center py-0 text-xs">
                    <AlertDescription>{getErrorMessage(error)}</AlertDescription>
                </Alert>
            </div>
        );
    }

    // ── Single-select ─────────────────────────────────────────────────────
    if (!isMultiSelect) {
        const selected = items.find((o) => o.value === value) ?? null;

        return (
            <Combobox
                items={items}
                itemToStringValue={(item) => item.label}
                value={selected}
                onValueChange={(next) => onChange(next ? next.value : "")}
            >
                <ComboboxInput
                    className={cn("w-full", className)}
                    placeholder={placeholder}
                    showClear={!!selected}
                />
                <ComboboxContent>
                    <ComboboxEmpty>No class found.</ComboboxEmpty>
                    <ComboboxList>
                        {(item: ClassOption) => (
                            <ComboboxItem key={item.value} value={item}>
                                {item.label}
                            </ComboboxItem>
                        )}
                    </ComboboxList>
                </ComboboxContent>
            </Combobox>
        );
    }

    // ── Multi-select ──────────────────────────────────────────────────────
    const selectedItems = items.filter((o) => (value as string[]).includes(o.value));

    return (
        <Combobox
            items={items}
            itemToStringValue={(item) => item.label}
            multiple
            value={selectedItems}
            onValueChange={(next) => onChange(next.map((item: ClassOption) => item.value))}
        >
            <ComboboxChips className={cn("w-full", className)}>
                <ComboboxValue>
                    {selectedItems.map((item) => (
                        <ComboboxChip key={item.value}>{item.label}</ComboboxChip>
                    ))}
                </ComboboxValue>
                <ComboboxChipsInput placeholder={placeholder} />
            </ComboboxChips>
            <ComboboxContent>
                <ComboboxEmpty>No class found.</ComboboxEmpty>
                <ComboboxList>
                    {(item: ClassOption) => (
                        <ComboboxItem key={item.value} value={item}>
                            {item.label}
                        </ComboboxItem>
                    )}
                </ComboboxList>
            </ComboboxContent>
        </Combobox>
    );
}
