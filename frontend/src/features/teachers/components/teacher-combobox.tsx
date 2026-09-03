/**
 * TeacherCombobox — reusable teacher selector built on Radix-only Combobox.
 *
 * Fetches its own options internally — zero prop drilling.
 * Stable TanStack Query key → 1 network call for N instances.
 */

"use client";

import { toast } from "sonner";
import * as React from "react";
import Link from "next/link";

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
import { useTeachers } from "../hooks/use-teachers";

interface Option {
    value: string;
    label: string;
}

// ─── Props ────────────────────────────────────────────────────────────────

export interface TeacherComboboxProps {
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

export function TeacherCombobox({
    value,
    onChange,
    placeholder = "Select a teacher...",
    className,
    isMultiSelect = false,
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

    // ── No items at all ──────────────────────────────────────────────────
    if (items.length === 0) {
        return (
            <Link href="/teachers/add" className="underline underline-offset-2">
                Invite teachers
            </Link>
        );
    }

    // ── Multi-select ────────────────────────────────────────────────────
    return (
        <Combobox
            items={items as Option[]}
            value={selectedOption}
            itemToStringValue={(i) => i.label}
            onValueChange={(v) => {
                if (v) {
                    onChange(isMultiSelect ? [v] : v);
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
                        <ComboboxItem key={i.value} value={i as Option}>
                            {i.label}
                        </ComboboxItem>
                    )}
                </ComboboxList>
            </ComboboxContent>
        </Combobox>
    );
}
