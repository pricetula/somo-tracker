/**
 * Combobox — Radix-only combobox built on Popover + Command.
 *
 * Dependencies: @radix-ui/react-popover, cmdk, lucide-react
 * No Base UI dependency anywhere.
 *
 * Supports:
 *  - Single-select (default)
 *  - Multi-select (`multiple` prop, checkbox-style toggling)
 *  - Controlled value / onValueChange
 *  - Keyboard navigation and search filtering (via cmdk)
 *  - Custom trigger via `renderTrigger` (for chips-based multi-select)
 *  - Placeholder and empty state
 */

"use client";

import * as React from "react";
import { Check, ChevronsUpDown } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
    Command,
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
} from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

// ─── Context ─────────────────────────────────────────────────────────────

interface ComboboxContextValue {
    multiple?: boolean;
    selectedValues: string[];
    onItemSelect: (value: string) => void;
}

const ComboboxContext = React.createContext<ComboboxContextValue | null>(null);

function useComboboxContext() {
    const ctx = React.useContext(ComboboxContext);
    if (!ctx) throw new Error("Combobox sub-components must be used within <Combobox>");
    return ctx;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function toValues(value: string | string[] | undefined, multiple: boolean): string[] {
    if (!value) return [];
    if (multiple) return value as string[];
    return (value as string) ? [value as string] : [];
}

function toLabel(
    selectedValues: string[],
    items: readonly { value: string; label: string }[],
    multiple: boolean
): string {
    if (selectedValues.length === 0) return "";
    if (!multiple || selectedValues.length === 1) {
        const match = items.find((i) => i.value === selectedValues[0]);
        return match?.label ?? "";
    }
    return `${selectedValues.length} selected`;
}

// ─── Root ─────────────────────────────────────────────────────────────────

export interface ComboboxProps {
    /** Display items — must have `value` and `label`. */
    items: readonly { value: string; label: string }[];
    /** Controlled selected value(s). */
    value?: string | string[];
    /** Called when selection changes. */
    onValueChange?: (value: string | string[]) => void;
    /** Enable multi-select (checkbox-style). */
    multiple?: boolean;
    /** Placeholder when nothing is selected. */
    placeholder?: string;
    /** Custom empty state message. */
    emptyText?: string;
    /** Class on the trigger button. */
    className?: string;
    /** Class on the popover content. */
    contentClassName?: string;
    /**
     * Custom trigger renderer for multi-select chips.
     * When provided, `renderTrigger` renders the trigger element and the
     * popover content (with search + items) is auto-rendered underneath.
     */
    renderTrigger?: (helpers: {
        selectedItems: { value: string; label: string }[];
        placeholder: string;
    }) => React.ReactNode;
}

export function Combobox({
    items,
    value,
    onValueChange,
    multiple = false,
    placeholder = "Select...",
    emptyText = "No items found.",
    className,
    contentClassName,
    renderTrigger,
}: ComboboxProps) {
    const [open, setOpen] = React.useState(false);
    const [search, setSearch] = React.useState("");

    const selectedValues = React.useMemo(() => toValues(value, multiple), [value, multiple]);

    const selectionLabel = React.useMemo(
        () => toLabel(selectedValues, items, multiple),
        [selectedValues, items, multiple]
    );

    const selectedItems = React.useMemo(
        () => items.filter((i) => selectedValues.includes(i.value)),
        [items, selectedValues]
    );

    const handleItemSelect = React.useCallback(
        (itemValue: string) => {
            if (multiple) {
                const current = selectedValues;
                const next = current.includes(itemValue)
                    ? current.filter((v) => v !== itemValue)
                    : [...current, itemValue];
                onValueChange?.(next);
            } else {
                const next = value === itemValue ? "" : itemValue;
                onValueChange?.(next);
                setOpen(false);
                setSearch("");
            }
        },
        [multiple, selectedValues, value, onValueChange]
    );

    const handleOpenChange = React.useCallback((newOpen: boolean) => {
        setOpen(newOpen);
        if (!newOpen) setSearch("");
    }, []);

    const ctx = React.useMemo<ComboboxContextValue>(
        () => ({
            multiple,
            selectedValues,
            onItemSelect: handleItemSelect,
        }),
        [multiple, selectedValues, handleItemSelect]
    );

    // ── Filtered items for display ─────────────────────────────────────
    const filtered = React.useMemo(() => {
        if (!search.trim()) return items;
        const q = search.toLowerCase();
        return items.filter((item) => item.label.toLowerCase().includes(q));
    }, [items, search]);

    // ── Default trigger (single-select button) ─────────────────────────
    const defaultTrigger = (
        <PopoverTrigger asChild>
            <Button
                variant="outline"
                role="combobox"
                aria-expanded={open}
                className={cn(
                    "h-9 w-full justify-between px-3 text-sm font-normal",
                    !selectionLabel && "text-muted-foreground",
                    className
                )}
            >
                <span className="truncate">{selectionLabel || placeholder}</span>
                <ChevronsUpDown className="ml-2 size-4 shrink-0 opacity-50" />
            </Button>
        </PopoverTrigger>
    );

    // ── Content (search + items list) ──────────────────────────────────
    const content = (
        <PopoverContent
            className={cn("w-(--radix-popover-trigger-width) p-0", contentClassName)}
            align="start"
            sideOffset={4}
        >
            <Command shouldFilter={false}>
                <CommandInput placeholder={`Search...`} value={search} onValueChange={setSearch} />
                <CommandList>
                    <CommandEmpty>{emptyText}</CommandEmpty>
                    <CommandGroup>
                        {filtered.map((item) => (
                            <CommandItem
                                key={item.value}
                                value={item.value}
                                onSelect={() => handleItemSelect(item.value)}
                            >
                                <span className="flex-1 truncate">{item.label}</span>
                                <Check
                                    className={cn(
                                        "ml-auto size-4 shrink-0",
                                        selectedValues.includes(item.value)
                                            ? "opacity-100"
                                            : "opacity-50"
                                    )}
                                />
                            </CommandItem>
                        ))}
                    </CommandGroup>
                </CommandList>
            </Command>
        </PopoverContent>
    );

    return (
        <ComboboxContext.Provider value={ctx}>
            <Popover open={open} onOpenChange={handleOpenChange}>
                {renderTrigger ? (
                    <PopoverTrigger asChild>
                        {renderTrigger({ selectedItems, placeholder })}
                    </PopoverTrigger>
                ) : (
                    defaultTrigger
                )}
                {content}
            </Popover>
        </ComboboxContext.Provider>
    );
}

// ─── Sub-components (for manual composition with custom children) ────────

export function ComboboxChips({
    children,
    className,
}: {
    children?: React.ReactNode;
    className?: string;
}) {
    const { selectedValues } = useComboboxContext();
    return (
        <Button
            variant="outline"
            role="combobox"
            className={cn(
                "flex h-auto min-h-9 w-full flex-wrap items-center gap-1 px-3 py-1.5 text-sm font-normal",
                className
            )}
        >
            {selectedValues.length > 0 && children}
            <ChevronsUpDown className="ml-auto size-4 shrink-0 opacity-50" />
        </Button>
    );
}

export function ComboboxChip({
    value,
    onRemove,
    children,
}: {
    value: string;
    onRemove?: (value: string) => void;
    children?: React.ReactNode;
}) {
    return (
        <span className="bg-secondary text-secondary-foreground inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-medium">
            {children}
            {onRemove && (
                <button
                    type="button"
                    onClick={(e) => {
                        e.stopPropagation();
                        onRemove(value);
                    }}
                    className="hover:text-foreground ml-0.5 inline-flex size-3.5 items-center justify-center rounded-sm opacity-60 transition-opacity hover:opacity-100"
                    aria-label={`Remove ${children}`}
                >
                    ×
                </button>
            )}
        </span>
    );
}
