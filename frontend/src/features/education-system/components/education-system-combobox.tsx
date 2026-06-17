"use client";

import * as React from "react";
import { Check, ChevronsUpDown, Globe } from "lucide-react";

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
import { useEducationSystems } from "@/hooks/use-education-systems";

export interface EducationSystemComboboxProps {
    /** Currently selected system ID (controlled). */
    value?: string;
    /** Called when a system is selected. */
    onValueChange: (value: string) => void;
    /** Optional placeholder text. */
    placeholder?: string;
    /** Optional className for the trigger button. */
    className?: string;
    /** Disable the combobox. */
    disabled?: boolean;
}

/**
 * Shadcn combobox for selecting an education system.
 *
 * Fetches systems via TanStack Query with `staleTime: Infinity` — the data
 * is loaded once per session and never re-fetched automatically.
 *
 * Usage:
 * ```tsx
 * const [value, setValue] = React.useState("");
 * <EducationSystemCombobox value={value} onValueChange={setValue} />
 * ```
 */
export function EducationSystemCombobox({
    value,
    onValueChange,
    placeholder = "Select education system…",
    className,
    disabled = false,
}: EducationSystemComboboxProps) {
    const [open, setOpen] = React.useState(false);
    const { data: systems = [], isLoading, isError } = useEducationSystems();

    const selected = systems.find((s) => s.id === value);

    return (
        <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
                <Button
                    variant="outline"
                    role="combobox"
                    aria-expanded={open}
                    disabled={disabled || isLoading || isError}
                    className={cn(
                        "w-full justify-between font-normal",
                        !selected && "text-muted-foreground",
                        className
                    )}
                >
                    <span className="flex items-center gap-2 truncate">
                        {isLoading ? (
                            <span className="text-muted-foreground">Loading…</span>
                        ) : isError ? (
                            <span className="text-destructive">Failed to load</span>
                        ) : selected ? (
                            <>{selected.name}</>
                        ) : (
                            <>{placeholder}</>
                        )}
                    </span>
                    <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                </Button>
            </PopoverTrigger>
            <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0">
                <Command>
                    <CommandInput placeholder="Search education systems…" />
                    <CommandList>
                        <CommandEmpty>No education system found.</CommandEmpty>
                        <CommandGroup>
                            {systems.map((system) => (
                                <CommandItem
                                    key={system.id}
                                    value={system.id}
                                    onSelect={(currentValue) => {
                                        onValueChange(currentValue === value ? "" : currentValue);
                                        setOpen(false);
                                    }}
                                >
                                    <Globe className="text-muted-foreground mr-2 h-4 w-4" />
                                    <span>{system.name}</span>
                                    <span className="text-muted-foreground ml-2 text-xs">
                                        {system.country_code}
                                    </span>
                                    <Check
                                        className={cn(
                                            "ml-auto h-4 w-4",
                                            value === system.id ? "opacity-100" : "opacity-0"
                                        )}
                                    />
                                </CommandItem>
                            ))}
                        </CommandGroup>
                    </CommandList>
                </Command>
            </PopoverContent>
        </Popover>
    );
}
