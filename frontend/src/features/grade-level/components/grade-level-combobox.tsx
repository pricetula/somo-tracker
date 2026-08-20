/**
 * GradeLevelCombobox — reusable grade level selector built on the shared Combobox.
 *
 * Fetches its own options from GRADE_LEVEL_LABELS — zero prop drilling.
 * Place in the grade-level feature so all consumers import from one place.
 */

"use client";

import { useEffect, useMemo, useRef } from "react";
import * as React from "react";

import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
} from "@/components/ui/combobox";
import { GRADE_LEVEL_LABELS } from "../types";

// ─── Props ────────────────────────────────────────────────────────────────

export interface GradeLevelComboboxProps {
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

export function GradeLevelCombobox({
    value,
    onChange,
    placeholder = "Select a grade level...",
    className,
    doPreselectFirstOption = false,
}: GradeLevelComboboxProps) {
    const items = useMemo(
        () =>
            Object.entries(GRADE_LEVEL_LABELS).map(([value, label]) => ({
                value,
                label,
            })),
        []
    );

    // ── Auto-preselect first option ──────────────────────────────────────
    const hasPreselected = useRef(false);
    useEffect(() => {
        if (!doPreselectFirstOption || items.length === 0 || hasPreselected.current) return;
        if (value) {
            hasPreselected.current = true;
            return;
        }
        hasPreselected.current = true;
        onChange(items[0].value);
    }, [doPreselectFirstOption, items, value, onChange]);

    return React.createElement(
        Combobox,
        { value, onValueChange: (v: string | null) => onChange(v as string), className },
        React.createElement(ComboboxInput, { placeholder }),
        React.createElement(
            ComboboxContent,
            null,
            React.createElement(
                ComboboxList,
                null,
                items.map((item) =>
                    React.createElement(
                        ComboboxItem,
                        { key: item.value, value: item.value },
                        item.label
                    )
                )
            )
        )
    );
}
