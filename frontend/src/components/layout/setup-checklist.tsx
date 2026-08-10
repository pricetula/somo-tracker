/**
 * SetupChecklist — shows onboarding progress for a new school.
 *
 * Checks which foundational configuration items are in place:
 * academic year, classes, students, curriculum, timetable.
 * Only renders when at least one item is incomplete.
 */

"use client";

import Link from "next/link";
import { CheckIcon, CircleIcon, TriangleAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useDashboardSetupProgress } from "@/features/dashboard/hooks/use-dashboard-summary";

export function SetupChecklist() {
    const { data: items, isLoading } = useDashboardSetupProgress();

    if (isLoading) {
        return null;
    }

    if (!items || items.every((item) => item.done)) {
        return null;
    }

    const doneCount = items.filter((i) => i.done).length;
    const totalCount = items.length;

  return (

      <Popover
      >
          <PopoverTrigger asChild>
              <Button
          variant="ghost"
          size="icon-lg"
          className="fixed z-20 right-4 top-28"
              >
                <TriangleAlert className="text-amber-600"/>
              </Button>
          </PopoverTrigger>
          <PopoverContent
              className="w-2xs p-0"

        >
        <section className="p-4">
            <h2 className="mb-1 text-lg font-medium">Getting Started</h2>
            <p className="text-muted-foreground mb-3 text-sm">
                {doneCount} of {totalCount} setup steps complete
            </p>
            <div className="space-y-2">
                {items.map((item) => (
                    <Link
                        key={item.id}
                        href={item.href}
                        className={`flex items-center gap-3 rounded-lg px-3 py-2 transition-colors ${
                            item.done ? "hover:bg-muted/50" : "bg-muted/30 hover:bg-muted/50"
                        }`}
                    >
                        {item.done ? (
                            <CheckIcon className="size-4 shrink-0 text-emerald-600" />
                        ) : (
                            <CircleIcon className="text-muted-foreground size-4 shrink-0" />
                        )}
                        <div className="min-w-0">
                            <p
                                className={`text-sm ${
                                    item.done ? "text-muted-foreground line-through" : "font-medium"
                                }`}
                            >
                                {item.label}
                            </p>
                            <p className="text-muted-foreground text-xs">{item.description}</p>
                        </div>
                    </Link>
                ))}
            </div>
        </section>

        </PopoverContent>
      </Popover>

    );
}
