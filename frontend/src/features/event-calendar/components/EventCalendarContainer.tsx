"use client";

import { TimelineBar } from "./TimelineBar";

export function EventCalendarContainer() {
    const year = new Date().getFullYear();

    return (
        <div className="space-y-4">
            <h1 className="text-2xl font-semibold">Annual Timeline</h1>
            <TimelineBar year={year} />
        </div>
    );
}
