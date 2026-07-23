/**
 * AcademicYearCombobox — reusable academic year selector built on the shared Combobox.
 *
 * Fetches its own options via useAcademicYears — zero prop drilling.
 * Place in the academic-terms feature so all consumers import from one place.
 */

"use client";

import { useEffect, useMemo } from "react";

import { Combobox } from "@/components/ui/combobox";
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
        if (!value && data?.items && data.items.length > 0) {
            const current = data.items.find((y) => y.is_current);
            if (current) {
                onChange(current.id);
            }
        }
    }, [data, value, onChange]);

    const items = useMemo(() => {
        if (!data?.items) return [];
        return data.items.map((y) => ({
            value: y.id,
            label: y.name,
        }));
    }, [data]);

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

    return (
        <Combobox
            items={items}
            value={value}
            onValueChange={(v) => onChange(v as string)}
            placeholder={placeholder}
            className={className}
            onCreateItem={onCreateItem}
        />
    );
}
