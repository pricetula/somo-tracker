/**
 * GradeLevelCombobox — reusable grade level selector built on the shared Combobox.
 *
 * Fetches its own options from GRADE_LEVEL_LABELS — zero prop drilling.
 * Place in the grade-level feature so all consumers import from one place.
 */

"use client";

import React from "react";

import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";
import { GRADE_LEVEL_LABELS } from "../types";

interface Option {
    value: string;
    label: string;
}

// ─── Props ────────────────────────────────────────────────────────────────
interface GradeLevelComboboxProps {
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

export function GradeLevelCombobox({
    value,
    onChange,
    placeholder = "Select a grade level...",
    className,
}: GradeLevelComboboxProps) {
    const items = React.useMemo<Option[]>(
        () =>
            Object.entries(GRADE_LEVEL_LABELS).map(([value, label]) => ({
                value,
                label,
            })),
        []
    );

    const selectedOption = React.useMemo(
        () => items.find((o) => o.value === value) || items[0],
        [items, value]
    );

    React.useEffect(() => {
        if (!value && selectedOption) {
            onChange(selectedOption.value);
        }
    }, [selectedOption, value, onChange]);

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
