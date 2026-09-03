"use client";

import * as React from "react";
import { format } from "date-fns";
import { CalendarIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

interface DatePickerProps {
    /** The selected value as a YYYY-MM-DD string (or empty string for no selection). */
    value: string;
    /** Called with the new YYYY-MM-DD string when the selection changes (or empty string when cleared). */
    onChange: (value: string) => void;
    /** Placeholder text when no date is selected. */
    placeholder?: string;
    /** Whether the date picker is disabled. */
    disabled?: boolean;
    /** Optional className to apply to the trigger button. */
    className?: string;
    /** Optional id for the trigger button. */
    id?: string;
}

function DatePicker({
    value,
    onChange,
    placeholder = "Pick a date",
    disabled = false,
    className,
    id,
}: DatePickerProps) {
    const [open, setOpen] = React.useState(false);

    // Convert the YYYY-MM-DD string to a Date for the calendar
    const date = value ? new Date(value + "T00:00:00") : undefined;

    const handleSelect = (selectedDate: Date | undefined) => {
        if (selectedDate) {
            onChange(format(selectedDate, "yyyy-MM-dd"));
        } else {
            onChange("");
        }
        setOpen(false);
    };

    return (
        <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger
                render={
                    <Button
                        id={id}
                        variant={"outline"}
                        data-empty={!date}
                        className={cn(
                            "data-[empty=true]:text-muted-foreground w-53 justify-between text-left font-normal",
                            className
                        )}
                    >
                        {date ? format(date, "PPP") : <span>{placeholder}</span>}
                        <CalendarIcon data-icon="inline-end" />
                    </Button>
                }
            />
            <PopoverContent className="w-auto p-0" align="start">
                <Calendar
                    mode="single"
                    selected={date}
                    onSelect={handleSelect}
                    required
                    autoFocus
                    disabled={disabled}
                    captionLayout="dropdown"
                />
            </PopoverContent>
        </Popover>
    );
}

export { DatePicker };
