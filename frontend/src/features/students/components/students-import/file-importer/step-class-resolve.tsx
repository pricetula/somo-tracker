"use client";

import * as React from "react";
import { Check, ChevronsUpDown, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
    Command,
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
} from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { cn } from "@/lib/utils";
import { useClassList } from "@/features/classes";
import type { Class } from "@/features/classes";
import type { ClassOption } from "@/features/classes/types";
import type { UnresolvedClassEntry } from "./types";
import {
    resolveClassStrings,
    mergeCountsIntoEntries,
    countClassOccurrences,
} from "./utils/class-resolver-utils";

interface StepClassResolveProps {
    rows: Record<string, string>[];
    classColumn: string | null;
    allHeaders: string[];
    onResolveComplete: (classMappings: Record<string, string>) => void;
    onBack: () => void;
    initialMappings?: Record<string, string>;
}

// ─── Class Resolver Row ───────────────────────────────────────────────────

function ClassResolverRow({
    entry,
    classOptions,
    onResolve,
}: {
    entry: UnresolvedClassEntry;
    classOptions: ClassOption[];
    onResolve: (rawString: string, classId: string | null) => void;
}) {
    const [open, setOpen] = React.useState(false);
    const [localResolvedId, setLocalResolvedId] = React.useState<string | null>(entry.resolved_id);

    const resolvedLabel = React.useMemo(() => {
        if (!localResolvedId) return null;
        return classOptions.find((c) => c.value === localResolvedId)?.label ?? null;
    }, [localResolvedId, classOptions]);

    const statusBadge = React.useMemo(() => {
        switch (entry.status) {
            case "matched":
                return (
                    <Badge
                        variant="default"
                        className="bg-emerald-500/10 text-[10px] text-emerald-600"
                    >
                        Auto
                    </Badge>
                );
            case "ambiguous":
                return (
                    <Badge
                        variant="outline"
                        className="border-amber-200 text-[10px] text-amber-600"
                    >
                        Review
                    </Badge>
                );
            case "unmatched":
                return (
                    <Badge variant="outline" className="text-destructive text-[10px]">
                        Unmatched
                    </Badge>
                );
        }
    }, [entry.status]);

    const handleSelect = React.useCallback(
        (value: string) => {
            const newId = value === "skip" ? null : value;
            setLocalResolvedId(newId);
            setOpen(false);
            onResolve(entry.raw_string, newId);
        },
        [entry.raw_string, onResolve]
    );

    return (
        <div className="flex items-center gap-3 px-2 py-1.5">
            {/* Raw string */}
            <div className="flex w-50 shrink-0 items-center gap-1.5">
                <span className="truncate">{entry.raw_string}</span>
                {entry.count > 1 && (
                    <span className="text-muted-foreground shrink-0 text-[10px]">
                        ({entry.count})
                    </span>
                )}
            </div>

            {/* Status */}
            <div className="flex w-20 shrink-0 items-center gap-1">{statusBadge}</div>

            {/* Resolver dropdown */}
            <div className="flex-1">
                {entry.status === "matched" && localResolvedId ? (
                    <span className="text-muted-foreground block truncate">{resolvedLabel}</span>
                ) : (
                    <Popover open={open} onOpenChange={setOpen}>
                        <PopoverTrigger asChild>
                            <Button
                                variant="outline"
                                role="combobox"
                                aria-expanded={open}
                                className={cn(
                                    "h-7 w-full justify-between px-2 text-xs font-normal",
                                    !localResolvedId && "text-muted-foreground"
                                )}
                            >
                                <span className="truncate">
                                    {resolvedLabel ??
                                        (entry.status === "unmatched"
                                            ? "No match — select class"
                                            : "Select class...")}
                                </span>
                                <ChevronsUpDown className="ml-1 size-3 shrink-0 opacity-50" />
                            </Button>
                        </PopoverTrigger>
                        <PopoverContent
                            className="w-(--radix-popover-trigger-width) p-0"
                            align="start"
                            sideOffset={4}
                        >
                            <Command shouldFilter={false}>
                                <CommandInput placeholder="Search classes..." />
                                <CommandList>
                                    <CommandEmpty>No classes found.</CommandEmpty>
                                    <CommandGroup>
                                        <CommandItem
                                            key="skip"
                                            value="skip"
                                            onSelect={() => handleSelect("skip")}
                                        >
                                            <span className="text-muted-foreground">
                                                Leave unmapped
                                            </span>
                                            {localResolvedId === null && (
                                                <Check className="ml-auto size-3.5 shrink-0 opacity-100" />
                                            )}
                                        </CommandItem>
                                        {classOptions.map((opt) => (
                                            <CommandItem
                                                key={opt.value}
                                                value={opt.value}
                                                onSelect={() => handleSelect(opt.value)}
                                            >
                                                <span className="flex-1 truncate">{opt.label}</span>
                                                {localResolvedId === opt.value && (
                                                    <Check className="ml-auto size-3.5 shrink-0 opacity-100" />
                                                )}
                                            </CommandItem>
                                        ))}
                                    </CommandGroup>
                                </CommandList>
                            </Command>
                        </PopoverContent>
                    </Popover>
                )}
            </div>
        </div>
    );
}

// ─── Main Component ───────────────────────────────────────────────────────

export function StepClassResolve({
    rows,
    classColumn,
    onResolveComplete,
    onBack,
    initialMappings,
}: StepClassResolveProps) {
    const { data: classListResult, isLoading: classesLoading } = useClassList();

    // Compute auto-mappings derived from class data
    const autoMappings = React.useMemo<Record<string, string>>(() => {
        if (classesLoading || !classListResult?.items || !classColumn) return {};

        const backendClasses = classListResult.items as unknown as Class[];
        const counts = countClassOccurrences(rows, classColumn);
        const rawStrings = new Set(counts.keys());
        if (rawStrings.size === 0) return {};

        const resolved = resolveClassStrings(rawStrings, backendClasses);
        const withCounts = mergeCountsIntoEntries(resolved, counts);

        const result: Record<string, string> = {};
        for (const entry of withCounts) {
            if (entry.status === "matched" && entry.resolved_id) {
                result[entry.raw_string] = entry.resolved_id;
            }
        }
        return result;
    }, [classListResult, classesLoading, classColumn, rows]);

    // User overrides: raw_string -> class_id
    const [userOverrides, setUserOverrides] = React.useState<Record<string, string>>({});

    // Effective mappings = autoMappings + initialMappings + userOverrides
    const classMappings = React.useMemo<Record<string, string>>(() => {
        const base = { ...autoMappings, ...initialMappings };
        return { ...base, ...userOverrides };
    }, [autoMappings, initialMappings, userOverrides]);

    const handleResolve = React.useCallback((rawString: string, classId: string | null) => {
        setUserOverrides((prev) => {
            const next = { ...prev };
            if (classId === null) {
                delete next[rawString];
            } else {
                next[rawString] = classId;
            }
            return next;
        });
    }, []);

    // Compute unresolved entries derived from props and current classMappings
    const unresolvedEntries: UnresolvedClassEntry[] = React.useMemo(() => {
        if (classesLoading || !classListResult?.items) return [];

        const backendClasses = classListResult.items as unknown as Class[];

        if (!classColumn) return [];

        const counts = countClassOccurrences(rows, classColumn);
        const rawStrings = new Set(counts.keys());

        if (rawStrings.size === 0) return [];

        const resolved = resolveClassStrings(rawStrings, backendClasses);
        const withCounts = mergeCountsIntoEntries(resolved, counts);

        // Apply initialMappings if resuming
        if (initialMappings) {
            for (const entry of withCounts) {
                if (initialMappings[entry.raw_string]) {
                    entry.resolved_id = initialMappings[entry.raw_string];
                    entry.status = "matched";
                }
            }
        }

        // Override with current classMappings state (user selections)
        for (const entry of withCounts) {
            const mappedId = classMappings[entry.raw_string];
            if (mappedId) {
                entry.resolved_id = mappedId;
                entry.status = "matched";
            }
        }

        return withCounts;
    }, [classListResult, classesLoading, classColumn, rows, initialMappings, classMappings]);

    const needsReview = React.useMemo(
        () => unresolvedEntries.filter((e) => e.status !== "matched" || !e.resolved_id),
        [unresolvedEntries]
    );

    const handleNext = React.useCallback(() => {
        onResolveComplete(classMappings);
    }, [classMappings, onResolveComplete]);

    const classOptions: ClassOption[] = React.useMemo(
        () => classListResult?.items ?? [],
        [classListResult]
    );

    // ─── Render ────────────────────────────────────────────────────────

    // If no class column was mapped, skip directly
    if (!classColumn) {
        return (
            <div className="space-y-4">
                <div>
                    <h3 className="font-medium">Class Resolution</h3>
                    <p className="text-muted-foreground mt-1 text-xs">
                        No class column was mapped, so all students will be created without a class
                        enrollment.
                    </p>
                </div>
                <div className="flex items-center justify-between">
                    <Button variant="ghost" size="sm" onClick={onBack}>
                        Back
                    </Button>
                    <Button size="sm" onClick={() => onResolveComplete({})}>
                        Next: Review & Import
                    </Button>
                </div>
            </div>
        );
    }

    if (classesLoading) {
        return (
            <div className="flex items-center justify-center py-8">
                <p className="text-muted-foreground">Loading classes...</p>
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <div>
                <h3 className="font-medium">Resolve Class/Stream Values</h3>
                <p className="text-muted-foreground mt-1 text-xs">
                    Review how each unique class value from your file maps to a school class.
                    &ldquo;Auto&rdquo; matches are high-confidence. Review items that need
                    attention.
                </p>
            </div>

            {/* Needs review alert */}
            {needsReview.length > 0 && (
                <Alert variant="destructive" className="py-2">
                    <AlertCircle className="size-4" />
                    <AlertTitle>{needsReview.length} need review</AlertTitle>
                    <AlertDescription>
                        {needsReview.filter((e) => e.status === "unmatched").length} unmatched,
                        {needsReview.filter((e) => e.status === "ambiguous").length} ambiguous
                        &mdash; please resolve below.
                    </AlertDescription>
                </Alert>
            )}

            {/* Class resolution rows */}
            {unresolvedEntries.length === 0 ? (
                <p className="text-muted-foreground py-4 text-center">
                    No class values found in the data.
                </p>
            ) : (
                <div className="rounded-md border">
                    {/* Fixed header */}
                    <div className="bg-muted/50 text-muted-foreground flex items-center gap-3 border-b px-2 py-1.5 text-[10px] font-medium tracking-wider uppercase">
                        <div className="w-50 shrink-0">Raw Value</div>
                        <div className="w-20 shrink-0">Status</div>
                        <div className="flex-1">Matched Class</div>
                    </div>
                    {/* Scrollable rows */}
                    <div className="max-h-80 overflow-y-auto">
                        {unresolvedEntries.map((entry) => (
                            <ClassResolverRow
                                key={entry.raw_string}
                                entry={entry}
                                classOptions={classOptions}
                                onResolve={handleResolve}
                            />
                        ))}
                    </div>
                </div>
            )}

            {/* Actions */}
            <div className="flex items-center justify-between pt-2">
                <Button variant="ghost" size="sm" onClick={onBack}>
                    Back
                </Button>
                <Button size="sm" onClick={handleNext}>
                    Next: Review & Import
                </Button>
            </div>
        </div>
    );
}
