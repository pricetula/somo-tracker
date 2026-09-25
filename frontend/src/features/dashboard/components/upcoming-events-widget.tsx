"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import { Skeleton } from "@/components/ui/skeleton";
import { format } from "date-fns";

type EventItem = {
    id: string;
    title: string;
    event_type: string;
    start_date: string;
    end_date: string;
    requires_attendance: boolean;
};

async function fetchEvents(): Promise<EventItem[]> {
    const from = format(new Date(), "yyyy-MM-dd");
    const to = format(new Date(Date.now() + 7 * 86400000), "yyyy-MM-dd");
    const res = await api.get<EventItem[]>(`/api/events?from=${from}&to=${to}`);
    return res;
}

export function UpcomingEventsWidget() {
    const { data, isLoading, isError } = useQuery({
        queryKey: ["events", "upcoming"],
        queryFn: fetchEvents,
        staleTime: 5 * 60 * 1000,
    });

    if (isLoading) {
        return (
            <div className="space-y-2">
                <Skeleton className="h-6 w-40" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-full" />
            </div>
        );
    }

    if (isError) {
        return <div className="text-destructive text-sm">Failed to load events.</div>;
    }

    if (!data || data.length === 0) {
        return (
            <div className="text-muted-foreground text-sm">
                No upcoming events in the next 7 days.
            </div>
        );
    }

    return (
        <article className="space-y-3">
            <h2 className="text-lg font-semibold">Upcoming events</h2>
            <ul className="space-y-2">
                {data.map((e) => (
                    <li key={e.id} className="flex items-center justify-between">
                        <div>
                            <div className="font-medium">{e.title}</div>
                            <div className="text-muted-foreground text-xs">
                                {e.event_type} • {e.start_date} to {e.end_date}
                                {e.requires_attendance && " • Attendance required"}
                            </div>
                        </div>
                    </li>
                ))}
            </ul>
        </article>
    );
}
