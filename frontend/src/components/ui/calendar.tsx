"use client";

import * as React from "react";
import { DayPicker } from "react-day-picker";

import { cn } from "@/lib/utils";

function Calendar({
    className,
    classNames,
    showOutsideDays = true,
    ...props
}: React.ComponentProps<typeof DayPicker>) {
    return (
        <DayPicker
            showOutsideDays={showOutsideDays}
            className={cn("p-2", className)}
            classNames={{
                root: "w-full",
                months: "flex flex-col gap-2",
                month: "flex flex-col gap-2",
                month_caption: "flex items-center justify-center py-1 relative",
                caption_label: "text-sm font-medium",
                nav: "flex items-center gap-1 absolute inset-x-0 top-0 justify-between",
                button_previous: cn(
                    "hover:bg-muted size-6 rounded-md inline-flex items-center justify-center transition-colors",
                    "disabled:pointer-events-none disabled:opacity-50"
                ),
                button_next: cn(
                    "hover:bg-muted size-6 rounded-md inline-flex items-center justify-center transition-colors",
                    "disabled:pointer-events-none disabled:opacity-50"
                ),
                month_grid: "w-full border-collapse",
                weekdays: "flex",
                weekday:
                    "text-muted-foreground flex size-8 items-center justify-center text-[0.625rem] font-medium",
                weeks: "flex flex-col",
                week: "flex mt-0.5 first:mt-0",
                day: cn(
                    "size-8 p-0 text-xs/relaxed font-normal",
                    "focus-within:relative focus-within:z-20"
                ),
                day_button: cn(
                    "hover:bg-muted hover:text-foreground size-full rounded-md inline-flex items-center justify-center transition-colors",
                    "focus-visible:ring-2 focus-visible:ring-ring/30 focus-visible:outline-none",
                    "aria-selected:bg-primary aria-selected:text-primary-foreground aria-selected:hover:bg-primary/90",
                    "aria-selected:focus-visible:ring-primary/30"
                ),
                outside: "text-muted-foreground/50 aria-selected:text-muted-foreground/50",
                disabled: "text-muted-foreground/50 pointer-events-none",
                hidden: "invisible",
                today: "after:content-[''] after:absolute after:bottom-0.5 after:left-1/2 after:-translate-x-1/2 after:size-0.5 after:rounded-full after:bg-primary",
                selected: "",
                range_start:
                    "aria-selected:bg-primary aria-selected:text-primary-foreground rounded-l-md",
                range_end:
                    "aria-selected:bg-primary aria-selected:text-primary-foreground rounded-r-md",
                range_middle:
                    "aria-selected:bg-primary/15 aria-selected:text-foreground rounded-none",
                ...classNames,
            }}
            {...props}
        />
    );
}

export { Calendar };
