"use client";

import * as React from "react";
import { DayPicker, UI, DayFlag, SelectionState } from "react-day-picker";

import { cn } from "@/lib/utils";
import { buttonVariants } from "@/components/ui/button";
import { ChevronLeft, ChevronRight } from "lucide-react";

export type CalendarProps = React.ComponentProps<typeof DayPicker>;

function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  ...props
}: CalendarProps) {
  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      className={cn("p-3", className)}
      classNames={{
        [UI.Months]: "flex flex-col sm:flex-row gap-2",
        [UI.Month]: "flex flex-col gap-4",
        [UI.MonthCaption]: "flex justify-center pt-1 relative items-center w-full",
        [UI.CaptionLabel]: "text-sm font-medium",
        [UI.Nav]: "flex items-center gap-1",
        [UI.PreviousMonthButton]: cn(
          buttonVariants({ variant: "outline" }),
          "size-7 bg-transparent p-0 opacity-50 hover:opacity-100 absolute left-1",
        ),
        [UI.NextMonthButton]: cn(
          buttonVariants({ variant: "outline" }),
          "size-7 bg-transparent p-0 opacity-50 hover:opacity-100 absolute right-1",
        ),
        [UI.Weekdays]: "flex",
        [UI.Weekday]:
          "text-muted-foreground rounded-2xl w-8 font-normal text-[0.8rem]",
        [UI.Week]: "flex w-full mt-2",
        [UI.Day]: cn(
          "relative p-0 text-center text-sm focus-within:relative focus-within:z-20",
          props.mode === "range"
            ? "[&:has(>[aria-selected])]:rounded-l-2xl [&:has(>[aria-selected])]:rounded-r-2xl"
            : "[&:has([aria-selected])]:rounded-2xl",
        ),
        [UI.DayButton]: cn(
          buttonVariants({ variant: "ghost" }),
          "size-8 p-0 font-normal aria-selected:opacity-100",
        ),
        [DayFlag.disabled]: "text-muted-foreground opacity-50 pointer-events-none",
        [SelectionState.selected]:
          "bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground",
        [DayFlag.today]: "bg-accent text-accent-foreground",
        [DayFlag.outside]:
          "text-muted-foreground opacity-50 aria-selected:text-muted-foreground",
        [DayFlag.hidden]: "invisible",
        [UI.MonthGrid]: "w-full border-collapse space-x-1",
        ...classNames,
      }}
      components={{
        Chevron: ({ orientation, className: chevronClassName }) => {
          if (orientation === "left") {
            return (
              <ChevronLeft
                className={cn("size-4", chevronClassName)}
                aria-hidden="true"
              />
            );
          }
          return (
            <ChevronRight
              className={cn("size-4", chevronClassName)}
              aria-hidden="true"
            />
          );
        },
      }}
      {...props}
    />
  );
}
Calendar.displayName = "Calendar";

export { Calendar };
