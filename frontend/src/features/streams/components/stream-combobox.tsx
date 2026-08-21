/**
 * StreamCombobox — reusable stream selector built on the shared Combobox.
 *
 * Fetches its own options via useStreamList — zero prop drilling.
 * Place in its own streams feature so all consumers import from one place.
 */

"use client";

import { toast } from "sonner";
import * as React from "react";
import Link from "next/link";
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
import { useStreamList } from "../hooks/use-streams";

interface Option {
    value: string;
    label: string;
}

// ─── Props ────────────────────────────────────────────────────────────────

export interface StreamComboboxProps {
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

export function StreamCombobox({
    value,
    onChange,
    placeholder = "Select a stream...",
    className,
}: StreamComboboxProps) {
    const { data, isLoading, isError, error } = useStreamList();

    const items = React.useMemo(() => {
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
        return <Skeleton className={className ?? "h-7 w-full"} />;
    }

    // ── No items at all (not just search miss) ───────────────────────────
    if (items.length === 0) {
        return (
            <Link href="/streams/add" className="underline underline-offset-2">
                Add one
            </Link>
        );
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
