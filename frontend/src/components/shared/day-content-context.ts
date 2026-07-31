"use client";

import * as React from "react";

/**
 * Shared context carrying the optional `dayContent` render prop injected into
 * each calendar day cell. Lives in its own module so both `Calendar` (provider)
 * and `CalendarDayButton` (consumer) reference the SAME context instance.
 */
export const DayContentContext = React.createContext<((date: Date) => React.ReactNode) | null>(
    null
);
