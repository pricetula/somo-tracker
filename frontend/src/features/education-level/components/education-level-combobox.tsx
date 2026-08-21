/**
 * EducationLevelCombobox — reusable education level selector built on the shared Combobox.
 *
 * Uses EDUCATION_LEVEL_LABELS as the single source of truth — zero duplication.
 * Place in the education-level feature so all consumers import from one place.
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
import { EDUCATION_LEVEL_LABELS } from "../types";

interface Option {
    value: string;
    label: string;
}

// ─── Props ────────────────────────────────────────────────────────────────

export interface EducationLevelComboboxProps {
    /** Currently selected education level (controlled). */
    value: string;
    /** Called when an education level is selected. */
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

export function EducationLevelCombobox({
    value,
    onChange,
    placeholder = "Select an education level...",
    className,
    doPreselectFirstOption = false,
}: EducationLevelComboboxProps) {
    const items = React.useMemo(
        () =>
            Object.entries(EDUCATION_LEVEL_LABELS).map(([value, label]) => ({
                value,
                label,
            })),
        []
    );

    // ── Auto-preselect first option ──────────────────────────────────────
    React.useEffect(() => {
        if (doPreselectFirstOption && items?.length && items.length > 0 && !value && onChange) {
            onChange(items[0].value);
        }
    }, [doPreselectFirstOption, items, value, onChange]);

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
