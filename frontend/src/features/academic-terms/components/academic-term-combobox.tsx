/**
 * AcademicTermCombobox — reusable academic term selector built on the shared Combobox.
 *
 * Fetches its own options via useAcademicTerms — zero prop drilling.
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
import { useAcademicTerms } from "../hooks/use-academic-terms";

interface Option {
    value: string;
    label: string;
}

// ─── Props ────────────────────────────────────────────────────────────────
export interface AcademicTermComboboxProps {
    /** Currently selected grade level (controlled). */
    value: string;
    /** Called when a grade level is selected. */
    onChange: (value: string) => void;
    /** Placeholder text when nothing is selected. */
    placeholder?: string;
    /** Optional outer container class. */
    className?: string;
}

// ─── Component ────────────────────────────────────────────────────────────

export function AcademicTermCombobox({
    value,
    onChange,
    placeholder = "Select a academic term...",
    className,
}: AcademicTermComboboxProps) {
    const { data, isLoading, isError, error } = useAcademicTerms();

    const items = React.useMemo(() => {
        console.log(data);
        if (!data?.items) return [];
        return data.items.map((t) => ({
            value: t.id,
            label: t.name,
        }));
    }, [data]);

    React.useEffect(() => {
        if (isError) {
            toast.error(getErrorMessage(error));
        }
    }, [isError, error]);

    const selectedOption = React.useMemo(
        () => items.find((o) => o.value === value) || items[0],
        [items, value]
    );

    React.useEffect(() => {
        if (!value && selectedOption) {
            onChange(selectedOption.value);
        }
    }, [selectedOption, value, onChange]);

    // ── Error state ──────────────────────────────────────────────────────
    if (isError) return null;

    // ── Loading state ─────────────────────────────────────────────────────
    if (isLoading) {
        return <Skeleton className={className ?? "h-9 w-full"} />;
    }

    return (
        <Combobox
            items={items as Option[]}
            value={selectedOption}
            itemToStringValue={(i) => i.label}
            onValueChange={(v) => {
                if (v) {
                    onChange(v.value);
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
