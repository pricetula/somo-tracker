"use client";

import { DAYS_OF_WEEK } from "../types";

interface TimetableHeaderProps {
    periods: Array<{ start_time: string; end_time: string; period_name: string }>;
}

export function TimetableHeader({}: TimetableHeaderProps) {
    return (
        <thead>
            <tr>
                <th className="bg-background/95 text-muted-foreground sticky left-0 z-10 w-32 px-2 py-2 text-left text-xs font-medium backdrop-blur-sm">
                    Period / Time
                </th>
                {DAYS_OF_WEEK.map(({ value: day, short }) => (
                    <th
                        key={day}
                        className="text-foreground w-48 min-w-48 px-2 py-2 text-center text-xs font-medium"
                    >
                        <div className="font-semibold">{short}</div>
                        <div className="text-muted-foreground text-xs">
                            {DAYS_OF_WEEK.find((d) => d.value === day)?.label}
                        </div>
                    </th>
                ))}
            </tr>
        </thead>
    );
}

interface PeriodSidebarProps {
    periods: Array<{
        start_time: string;
        end_time: string;
        period_name: string;
        is_break: boolean;
    }>;
}

export function PeriodSidebar({ periods }: PeriodSidebarProps) {
    return (
        <tbody>
            {periods.map((period) => (
                <tr key={period.period_name}>
                    <td className="bg-background/95 border-border sticky left-0 z-10 w-32 border-r px-2 py-2 text-left text-xs font-medium backdrop-blur-sm">
                        <div
                            className={
                                period.is_break ? "text-muted-foreground" : "text-foreground"
                            }
                        >
                            {period.period_name}
                        </div>
                        <div className="text-muted-foreground text-xs">
                            {formatTime(period.start_time)} - {formatTime(period.end_time)}
                        </div>
                        {period.is_break && (
                            <span className="bg-muted text-muted-foreground mt-0.5 inline-block rounded px-1.5 py-0.5 text-xs">
                                Break
                            </span>
                        )}
                    </td>
                </tr>
            ))}
        </tbody>
    );
}

function formatTime(time: string): string {
    // time format: "HH:MM:SS" or "HH:MM"
    const [hours, minutes] = time.split(":");
    const h = parseInt(hours, 10);
    const m = minutes.slice(0, 2);
    const ampm = h >= 12 ? "PM" : "AM";
    const displayHour = h % 12 || 12;
    return `${displayHour}:${m} ${ampm}`;
}
