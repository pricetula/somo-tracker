"use client";

/**
 * StepColumnMapping — map file columns to invitation fields.
 * Matches the student import StepColumnMapping pattern (Popover/Command combobox).
 */

import * as React from "react";
import { ArrowRight, Check, ChevronsUpDown } from "lucide-react";
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
import { cn } from "@/lib/utils";
import { SMART_MATCH_DICT, TARGET_FIELDS } from "./types";

interface StepColumnMappingProps {
    headers: string[];
    onMappingComplete: (mappings: Record<string, string | string[]>) => void;
    onBack: () => void;
    initialMappings?: Record<string, string | string[]>;
}

// ─── Smart matching helper ────────────────────────────────────────────────

function smartMatchTarget(headers: string[], targetKey: string): string | null {
    const variants = SMART_MATCH_DICT[targetKey];
    if (!variants) return null;

    const lowerHeaders = headers.map((h) => h.toLowerCase().trim());

    // Exact match first
    for (const variant of variants) {
        const idx = lowerHeaders.indexOf(variant.toLowerCase());
        if (idx !== -1) return headers[idx];
    }

    // Partial/substring match
    for (const variant of variants) {
        const vLower = variant.toLowerCase();
        const idx = lowerHeaders.findIndex((h) => h.includes(vLower) || vLower.includes(h));
        if (idx !== -1) return headers[idx];
    }

    return null;
}

// ─── Component ────────────────────────────────────────────────────────────

export function StepColumnMapping({
    headers,
    onMappingComplete,
    onBack,
    initialMappings,
}: StepColumnMappingProps) {
    const computedInitial = React.useMemo(() => {
        if (initialMappings) return initialMappings;

        const autoMapped: Record<string, string | string[]> = {};

        for (const field of TARGET_FIELDS) {
            const match = smartMatchTarget(headers, field.target_key);
            if (match) {
                autoMapped[field.target_key] = match;
            }
        }

        // Special handling for full_name: check if name is split across columns
        if (!autoMapped.full_name) {
            const firstNameIdx = headers.findIndex(
                (h) =>
                    h.toLowerCase().includes("first") || h.toLowerCase().includes("jina la kwanza")
            );
            const lastNameIdx = headers.findIndex(
                (h) =>
                    h.toLowerCase().includes("last") ||
                    h.toLowerCase().includes("surname") ||
                    h.toLowerCase().includes("jina la familia")
            );

            const parts: string[] = [];
            if (firstNameIdx !== -1) parts.push(headers[firstNameIdx]);
            if (lastNameIdx !== -1) parts.push(headers[lastNameIdx]);

            if (parts.length >= 2) {
                autoMapped.full_name = parts;
            }
        }

        return autoMapped;
    }, [headers, initialMappings]);

    const [mappings, setMappings] =
        React.useState<Record<string, string | string[]>>(computedInitial);

    const [openPopover, setOpenPopover] = React.useState<string | null>(null);

    const toggleHeaderForTarget = React.useCallback((targetKey: string, header: string) => {
        setMappings((prev) => {
            const current = prev[targetKey];

            if (targetKey === "full_name") {
                const currentArr = Array.isArray(current) ? current : current ? [current] : [];
                const next = currentArr.includes(header)
                    ? currentArr.filter((h) => h !== header)
                    : [...currentArr, header];
                return { ...prev, full_name: next.length > 0 ? next : header };
            }

            // Single-select: toggle off if same header
            if (current === header) {
                const { [targetKey]: _, ...rest } = prev;
                return rest;
            }
            return { ...prev, [targetKey]: header };
        });
    }, []);

    const isFullNameSelected = React.useCallback(
        (header: string) => {
            const val = mappings.full_name;
            if (Array.isArray(val)) return val.includes(header);
            return val === header;
        },
        [mappings.full_name]
    );

    const isSelected = React.useCallback(
        (targetKey: string, header: string) => {
            if (targetKey === "full_name") return isFullNameSelected(header);
            return mappings[targetKey] === header;
        },
        [mappings, isFullNameSelected]
    );

    // email must be mapped — gatekeeper
    const canProceed = React.useMemo(() => {
        const email = mappings.email;
        if (!email) return false;
        if (Array.isArray(email)) return email.length > 0;
        return email.length > 0;
    }, [mappings.email]);

    const handleNext = React.useCallback(() => {
        onMappingComplete(mappings);
    }, [mappings, onMappingComplete]);

    const getMappingDisplay = React.useCallback(
        (targetKey: string): string => {
            const val = mappings[targetKey];
            if (!val) return "";
            if (Array.isArray(val)) return val.join(" + ");
            return val;
        },
        [mappings]
    );

    // ─── Render ────────────────────────────────────────────────────────

    return (
        <div className="space-y-6">
            <div>
                <h3 className="font-medium">Map File Columns</h3>
                <p className="text-muted-foreground mt-1 text-xs">
                    Match your file columns to the invitation fields below. Only &ldquo;Email&rdquo;
                    is required.
                </p>
            </div>

            {/* Column listing */}
            <div className="space-y-3">
                {TARGET_FIELDS.map((field) => (
                    <div
                        key={field.target_key}
                        className="grid grid-cols-[1fr_auto_1fr] items-center gap-3"
                    >
                        {/* Target field label */}
                        <div className="min-w-0">
                            <span className="font-medium">{field.label}</span>
                            {field.required && (
                                <span className="text-destructive ml-1 text-xs">*</span>
                            )}
                        </div>

                        {/* Arrow */}
                        <ArrowRight className="text-muted-foreground size-3.5 shrink-0" />

                        {/* Mapping selector */}
                        <Popover
                            open={openPopover === field.target_key}
                            onOpenChange={(open) => setOpenPopover(open ? field.target_key : null)}
                        >
                            <PopoverTrigger asChild>
                                <Button
                                    variant="outline"
                                    role="combobox"
                                    aria-expanded={openPopover === field.target_key}
                                    className={cn(
                                        "h-9 w-full justify-between px-3 font-normal",
                                        !mappings[field.target_key] && "text-muted-foreground"
                                    )}
                                >
                                    <span className="truncate">
                                        {getMappingDisplay(field.target_key) || "Select column..."}
                                    </span>
                                    <ChevronsUpDown className="ml-2 size-4 shrink-0 opacity-50" />
                                </Button>
                            </PopoverTrigger>
                            <PopoverContent
                                className="w-(--radix-popover-trigger-width) p-0"
                                align="start"
                                sideOffset={4}
                            >
                                <Command shouldFilter={false}>
                                    <CommandInput placeholder="Search columns..." />
                                    <CommandList>
                                        <CommandEmpty>No columns found.</CommandEmpty>
                                        <CommandGroup>
                                            {headers.map((header) => (
                                                <CommandItem
                                                    key={header}
                                                    value={header}
                                                    onSelect={() =>
                                                        toggleHeaderForTarget(
                                                            field.target_key,
                                                            header
                                                        )
                                                    }
                                                >
                                                    <span className="flex-1 truncate">
                                                        {header}
                                                    </span>
                                                    <Check
                                                        className={cn(
                                                            "ml-auto size-4 shrink-0",
                                                            isSelected(field.target_key, header)
                                                                ? "opacity-100"
                                                                : "opacity-0"
                                                        )}
                                                    />
                                                </CommandItem>
                                            ))}
                                        </CommandGroup>
                                    </CommandList>
                                </Command>
                            </PopoverContent>
                        </Popover>
                    </div>
                ))}
            </div>

            {/* Multi-column indicator for full_name */}
            {Array.isArray(mappings.full_name) && mappings.full_name.length > 1 && (
                <div className="bg-muted/50 text-muted-foreground rounded-md p-3 text-xs">
                    Full name will be assembled from:{" "}
                    <span className="text-foreground font-medium">
                        {mappings.full_name.join(" + ")}
                    </span>
                </div>
            )}

            {/* Unmapped headers warning */}
            {headers.length > Object.keys(mappings).length && (
                <p className="text-muted-foreground text-xs">
                    {headers.length - Object.keys(mappings).length} column(s) unmapped — they will
                    be ignored.
                </p>
            )}

            {/* Action buttons */}
            <div className="flex items-center justify-between">
                <Button variant="ghost" size="sm" onClick={onBack}>
                    Back
                </Button>
                <Button size="sm" onClick={handleNext} disabled={!canProceed}>
                    Next: Review Records
                </Button>
            </div>
        </div>
    );
}
