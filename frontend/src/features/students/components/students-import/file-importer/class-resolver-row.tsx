"use client";

import { Check, ChevronsUpDown } from "lucide-react";
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
import { cn } from "@/lib/utils";
import { type ClassOption } from "@/features/classes/types";
import { type UnresolvedClassEntry } from "./types";
import * as React from "react";

export function ClassResolverRow({
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
                        <PopoverTrigger>
                            <Button
                                variant="outline"
                                role="combobox"
                                aria-expanded={open}
                                className={cn(
                                    "h-7 w-full justify-between px-2 font-normal",
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
                            className="w-(--base-ui-popover-trigger-width) p-0"
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
