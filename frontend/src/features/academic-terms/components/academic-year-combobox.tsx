/**
 * AcademicYearCombobox — reusable academic year selector built on the shared Combobox.
 *
 * Fetches its own options via useAcademicYears — zero prop drilling.
 * Place in the academic-terms feature so all consumers import from one place.
 */

"use client";

import { useEffect, useMemo, useCallback } from "react";
import * as React from "react";

import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { getErrorMessage } from "@/lib/errors";
import { useAcademicYears } from "../hooks/use-academic-terms";

// ─── Props ────────────────────────────────────────────────────────────────

export interface AcademicYearComboboxProps {
    /** Currently selected academic year ID (controlled). */
    value: string;
    /** Called when an academic year is selected. */
    onChange: (value: string) => void;
    /** Placeholder text when nothing is selected. */
    placeholder?: string;
    /** Optional outer container class. */
    className?: string;
    /**
     * When search yields no results, shows a "Create" option.
     * If omitted, no create option is shown.
     */
    onCreateItem?: (search: string) => void;
}

// ─── Constants ────────────────────────────────────────────────────────────

const CREATE_ITEM_VALUE = "__create__";

// ─── Component ────────────────────────────────────────────────────────────

export function AcademicYearCombobox({
    value,
    onChange,
    placeholder = "Select an academic year...",
    className,
    onCreateItem,
}: AcademicYearComboboxProps) {
    const { data, isLoading, isError, error } = useAcademicYears();

    // Auto-select the current academic year when data loads and no value is set
    useEffect(() => {
        if (!value && data?.data && data.data.length > 0) {
            const current = data.data.find((y) => y.is_current);
            if (current) {
                onChange(current.id);
            }
        }
    }, [data, value, onChange]);

    const items = useMemo(() => {
        if (!data?.data) return [];
        return data.data.map((y) => ({
            value: y.id,
            label: y.name,
        }));
    }, [data]);

    // ── Handle create item selection ──────────────────────────────────────
    const handleValueChange = useCallback(
        (newValue: string | null) => {
            if (newValue === CREATE_ITEM_VALUE && onCreateItem) {
                onCreateItem("");
            } else {
                onChange(newValue as string);
            }
        },
        [onChange, onCreateItem]
    );

    // ── Loading state ─────────────────────────────────────────────────────
    if (isLoading) {
        return <Skeleton className={className ?? "h-9 w-full"} />;
    }

    // ── Error state ──────────────────────────────────────────────────────
    if (isError) {
        return (
            <Alert variant="destructive" className="h-9 items-center py-0 text-xs">
                <AlertDescription>{getErrorMessage(error)}</AlertDescription>
            </Alert>
        );
    }

    return React.createElement(
        Combobox,
        { value, onValueChange: handleValueChange, className },
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
                ),
                onCreateItem &&
                    React.createElement(
                        ComboboxItem,
                        {
                            value: CREATE_ITEM_VALUE,
                            className: "bg-muted/50 text-muted-foreground italic",
                        },
                        "Create new academic year..."
                    ),
                React.createElement(ComboboxEmpty, null, "No academic years found")
            )
        )
    );
}
