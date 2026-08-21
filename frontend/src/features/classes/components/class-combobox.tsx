/**
 * ClassCombobox — reusable class selector built on Radix-only Combobox.
 *
 * Fetches its own options internally — zero prop drilling.
 * Stable TanStack Query key → 1 network call for N instances.
 * No React Context — each instance is fully isolated.
 */

"use client";

import * as React from "react";
import Link from "next/link";

import { toast } from "sonner";
import {
    Combobox,
    ComboboxChip,
    ComboboxChips,
    ComboboxChipsInput,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxInput,
    ComboboxItem,
    ComboboxList,
    ComboboxValue,
} from "@/components/ui/combobox";
import { Skeleton } from "@/components/ui/skeleton";
import { getErrorMessage } from "@/lib/errors";
import { useClassList } from "../hooks/use-classes";
import { ClassOption } from "../types";

// ─── Props ────────────────────────────────────────────────────────────────

export interface ClassComboboxProps {
    /** Currently selected grade level (controlled). */
    value: string;
    /** Called when a grade level is selected. */
    onChange: (value: string | string[]) => void;
    /** Placeholder text when nothing is selected. */
    placeholder?: string;
    /** Optional outer container class. */
    className?: string;
    /** Allow selecting multiple teachers (default: false). */
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

    const items = React.useMemo(() => data?.items || [], [data]);

    const selectedOption = React.useMemo(
        () => items.find((o) => o.value === value) || items[0],
        [items, value]
    );

    React.useEffect(() => {
        if (isError) {
            toast.error(getErrorMessage(error));
        }
    }, [isError, error]);

    React.useEffect(() => {
        if (!value && selectedOption) {
            onChange(selectedOption.value);
        }
    }, [selectedOption, value, onChange]);

    // ── Error state ──────────────────────────────────────────────────────
    if (isError) return null;

    // ── Loading state ─────────────────────────────────────────────────────
    if (isLoading) {
        return <Skeleton className="h-9 w-full" />;
    }

    // ── No items at all (not just search miss) ───────────────────────────
    if (items.length === 0) {
        return (
            <Link href="/classes/add" className="underline underline-offset-2">
                Add class
            </Link>
        );
    }

    // ── Multi-select ────────────────────────────────────────────────────
    return (
        <Combobox
            items={items as ClassOption[]}
            value={selectedOption}
            itemToStringValue={(i) => i.label}
            onValueChange={(v) => {
                if (v) {
                    const id = v.value;
                    onChange(id);
                }
            }}
            multiple={isMultiSelect}
        >
            {isMultiSelect ? (
                <ComboboxChips>
                    <ComboboxValue>
                        {(value as string[]).map((item) => (
                            <ComboboxChip key={item}>{item}</ComboboxChip>
                        ))}
                    </ComboboxValue>
                    <ComboboxChipsInput placeholder="Add framework" />
                </ComboboxChips>
            ) : (
                <ComboboxInput placeholder={placeholder} className={className} />
            )}

            <ComboboxContent>
                <ComboboxEmpty>No items found.</ComboboxEmpty>
                <ComboboxList>
                    {(i) => (
                        <ComboboxItem key={i.value} value={i as ClassOption}>
                            {i.label}
                        </ComboboxItem>
                    )}
                </ComboboxList>
            </ComboboxContent>
        </Combobox>
    );
}
