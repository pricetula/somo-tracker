/**
 * AcademicYearCombobox — reusable academic year selector built on the shared Combobox.
 *
 * Fetches its own options via useAcademicYears — zero prop drilling.
 * Place in the academic-terms feature so all consumers import from one place.
 */

"use client";

import React from "react";

import { toast } from "sonner";
import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";
import { Skeleton } from "@/components/ui/skeleton";
import { getErrorMessage } from "@/lib/errors";
import { useAcademicYears } from "../../academic-terms/hooks/use-academic-terms";

interface Option {
    value: string;
    label: string;
}

// ─── Props ────────────────────────────────────────────────────────────────
export interface AcademicYearComboboxProps {
    /** Currently selected grade level (controlled). */
    value: string;
    /** Called when a grade level is selected. */
    onChange: (value: string) => void;
    /** Placeholder text when nothing is selected. */
    placeholder?: string;
    /** Optional outer container class. */
    className?: string;
    /**
     * When true, automatically selects the first option if no value is set.
     * Defaults to false.
     */
    doPreselectFirstOption?: boolean;
}

// ─── Component ────────────────────────────────────────────────────────────

export function AcademicYearCombobox({
    value,
    onChange,
    placeholder = "Select a academic year...",
    className,
    doPreselectFirstOption = false,
}: AcademicYearComboboxProps) {
    const { data, isLoading, isError, error } = useAcademicYears();

    const items = React.useMemo(() => {
        if (!data?.data) return [];
        return data.data.map((t) => ({
            value: t.id,
            label: t.name,
        }));
    }, [data]);

    // ── Auto-preselect first option ──────────────────────────────────────
    React.useEffect(() => {
        if (doPreselectFirstOption && items?.length && items.length > 0 && !value && onChange) {
            onChange(items[0].value);
        }
    }, [doPreselectFirstOption, items, value, onChange]);

    React.useEffect(() => {
        if (isError) {
            toast.error(getErrorMessage(error));
        }
    }, [isError, error]);

    // ── Error state ──────────────────────────────────────────────────────
    if (isError) return null;

    // ── Loading state ─────────────────────────────────────────────────────
    if (isLoading) {
        return <Skeleton className={className ?? "h-9 w-full"} />;
    }

    return (
        <Combobox
            items={items as Option[]}
            value={value}
            itemToStringValue={(i) => i.label}
            onValueChange={(v) => {
                if (v) {
                    onChange(v);
                }
            }}
        >
            <ComboboxInput placeholder={placeholder} className={className} />
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
