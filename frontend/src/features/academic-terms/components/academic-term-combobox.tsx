/**
 * AcademicTermCombobox — reusable academic term selector built on the shared Combobox.
 *
 * Fetches its own options via useAcademicTerms — zero prop drilling.
 * Place in the academic-terms feature so all consumers import from one place.
 */

"use client";

import { useEffect, useMemo, useRef } from "react";

import { Combobox } from "@/components/ui/combobox";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { getErrorMessage } from "@/lib/errors";
import { useAcademicTerms } from "../hooks/use-academic-terms";

// ─── Props ────────────────────────────────────────────────────────────────

export interface AcademicTermComboboxProps {
    /** Currently selected academic term ID (controlled). */
    value: string;
    /** Called when an academic term is selected. */
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
    /**
     * When true, automatically selects the first option if no value is set.
     * Defaults to false.
     */
    doPreselectFirstOption?: boolean;
}

// ─── Component ────────────────────────────────────────────────────────────

export function AcademicTermCombobox({
    value,
    onChange,
    placeholder = "Select an academic term...",
    className,
    onCreateItem,
    doPreselectFirstOption = false,
}: AcademicTermComboboxProps) {
    const { data, isLoading, isError, error } = useAcademicTerms();

    const items = useMemo(() => {
        if (!data?.items) return [];
        return data.items.map((t) => ({
            value: t.id,
            label: t.name,
        }));
    }, [data]);

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
