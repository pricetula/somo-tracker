/**
 * Combobox for selecting from backend-driven academic reference data.
 *
 * Uses Popover + Command (shadcn combobox pattern) with search filtering.
 */

"use client";

import * as React from "react";
import { Check, ChevronsUpDown } from "lucide-react";
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

export interface ComboboxOption {
    value: string;
    label: string;
    isCurrent?: boolean;
}

export interface AcademicComboboxProps {
    options: ComboboxOption[];
    value: string;
    onValueChange: (value: string) => void;
    placeholder: string;
    searchPlaceholder?: string;
    emptyText?: string;
    loading?: boolean;
    disabled?: boolean;
}

export function AcademicCombobox({
    options,
    value,
    onValueChange,
    placeholder,
    searchPlaceholder = "Search...",
    emptyText = "No results found.",
    loading = false,
    disabled = false,
}: AcademicComboboxProps) {
    const [open, setOpen] = React.useState(false);

    const selectedLabel = React.useMemo(() => {
        const option = options.find((o) => o.value === value);
        return option ? option.label : placeholder;
    }, [options, value, placeholder]);

    return (
        <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
                <Button
                    variant="outline"
                    role="combobox"
                    aria-expanded={open}
                    disabled={disabled || loading}
                    className="w-full justify-between text-xs font-normal"
                >
                    {loading ? (
                        <span className="text-muted-foreground">Loading...</span>
                    ) : (
                        <span className={cn(!value && "text-muted-foreground")}>
                            {selectedLabel}
                        </span>
                    )}
                    <ChevronsUpDown className="ml-2 size-3.5 shrink-0 opacity-50" />
                </Button>
            </PopoverTrigger>
            <PopoverContent className="w-[--radix-popover-trigger-width] p-0">
                <Command>
                    <CommandInput placeholder={searchPlaceholder} />
                    <CommandList>
                        <CommandEmpty>{emptyText}</CommandEmpty>
                        <CommandGroup>
                            {options.map((option) => (
                                <CommandItem
                                    key={option.value}
                                    value={option.value}
                                    onSelect={(currentValue) => {
                                        onValueChange(currentValue === value ? "" : currentValue);
                                        setOpen(false);
                                    }}
                                >
                                    <Check
                                        className={cn(
                                            "mr-2 size-3.5",
                                            value === option.value ? "opacity-100" : "opacity-0"
                                        )}
                                    />
                                    <span>{option.label}</span>
                                    {option.isCurrent && (
                                        <span className="text-muted-foreground ml-2 text-[0.625rem]">
                                            (current)
                                        </span>
                                    )}
                                </CommandItem>
                            ))}
                        </CommandGroup>
                    </CommandList>
                </Command>
            </PopoverContent>
        </Popover>
    );
}
