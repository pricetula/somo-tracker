"use client";

import * as React from "react";
import { format } from "date-fns";
import { CalendarIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

export type DatePickerValue = Date | undefined;

interface DatePickerProps {
    value: DatePickerValue;
    onChange: (date: DatePickerValue) => void;
    disabled?: boolean;
    minDate?: Date;
    maxDate?: Date;
    placeholder?: string;
}

export function DatePicker({
    value,
    onChange,
    disabled,
    minDate,
    maxDate,
    placeholder = "Pick a date",
}: DatePickerProps) {
    const [open, setOpen] = React.useState(false);

    return (
        <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
                <Button
                    variant="outline"
                    disabled={disabled}
                    className={cn(
                        "w-full justify-start text-left font-normal",
                        !value && "text-muted-foreground"
                    )}
                >
                    <CalendarIcon className="mr-2 h-4 w-4 shrink-0" />
                    {value ? format(value, "PP") : <span>{placeholder}</span>}
                </Button>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-0" align="start">
                <Calendar
                    mode="single"
                    selected={value}
                    onSelect={(date) => {
                        onChange(date);
                        setOpen(false);
                    }}
                    disabled={(date: Date) => {
                        if (minDate && date < minDate) return true;
                        if (maxDate && date > maxDate) return true;
                        return false;
                    }}
                    autoFocus
                />
            </PopoverContent>
        </Popover>
    );
}
