"use client";

import * as React from "react";
import { useAcademicCalendar } from "@/features/calendar/hooks/use-academic-calendar";
import type { CalendarState } from "@/features/calendar/types";

/**
 * Evaluate the current academic calendar state using the Decision Tree
 * from the system architecture spec.
 *
 * CASE 1: API returns null or no periods → State A (setup form)
 * CASE 2: current_date within [first_start, final_end] → hidden (unlock dashboard)
 * CASE 3: current_date < first_start → hidden + prep-mode alert
 * CASE 4: current_date > final_end → State A (next-cycle form)
 */
export function useCalendarEvaluator(): CalendarState {
  const { data, isLoading } = useAcademicCalendar();

  return React.useMemo(() => {
    if (isLoading) return { type: "loading" };

    // CASE 1: Null or no periods → mount form
    if (!data || !data.periods || data.periods.length === 0) {
      return { type: "form", mode: "setup" };
    }

    const now = new Date();
    now.setHours(0, 0, 0, 0);

    const firstStart = new Date(data.periods[0].start_date);
    const finalPeriod = data.periods.find((p) => p.is_final);
    // If no explicit is_final, use the last period
    const finalEnd = finalPeriod
      ? new Date(finalPeriod.end_date)
      : new Date(data.periods[data.periods.length - 1].end_date);

    firstStart.setHours(0, 0, 0, 0);
    finalEnd.setHours(23, 59, 59, 999);

    // CASE 2: Within the active range → collapse form, unlock dashboard
    if (now >= firstStart && now <= finalEnd) {
      return { type: "hidden" };
    }

    // CASE 3: Before first start → prep mode alert
    if (now < firstStart) {
      return { type: "hidden", alert: "prep-mode" };
    }

    // CASE 4: After final end → form for next cycle
    return { type: "form", mode: "next-cycle" };
  }, [data, isLoading]);
}

/**
 * Render a minimalist alert strip for prep-mode.
 * Shown when the current date is before the academic year's first start.
 */
export function PrepModeAlert() {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-800/30 dark:bg-amber-950/30 dark:text-amber-300">
      <span className="text-base" role="img" aria-label="calendar">
        📅
      </span>
      <span>
        System in preparation mode for upcoming year. Calendar is configured and
        ready.
      </span>
    </div>
  );
}
