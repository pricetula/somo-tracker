/**
 * EducationLevelCombobox — reusable education level selector built on the shared Combobox.
 *
 * Uses EDUCATION_LEVEL_LABELS as the single source of truth — zero duplication.
 * Place in the education-level feature so all consumers import from one place.
 */

"use client";

import { useMemo } from "react";

import { Combobox } from "@/components/ui/combobox";
import { EDUCATION_LEVEL_LABELS } from "../types";

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
}

// ─── Component ────────────────────────────────────────────────────────────

export function EducationLevelCombobox({
    value,
    onChange,
    placeholder = "Select an education level...",
    className,
}: EducationLevelComboboxProps) {
    const items = useMemo(
        () =>
            Object.entries(EDUCATION_LEVEL_LABELS).map(([value, label]) => ({
                value,
                label,
            })),
        []
    );

    return (
        <Combobox
            items={items}
            value={value}
            onValueChange={(v) => onChange(v as string)}
            placeholder={placeholder}
            className={className}
        />
    );
}
