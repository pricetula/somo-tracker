import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type UpcomingEvent = {
    title: string;
    event_type: string;
    start_date: string;
    end_date: string;
    requires_attendance: boolean;
};

const events: UpcomingEvent[] = [
    {
        title: "Mid-term exams",
        event_type: "EXAM",
        start_date: "2026-10-20",
        end_date: "2026-10-24",
        requires_attendance: true,
    },
    {
        title: "Inter-House Sports Day",
        event_type: "SPORTS",
        start_date: "2026-10-18",
        end_date: "2026-10-18",
        requires_attendance: false,
    },
    {
        title: "Parent Teacher Meeting",
        event_type: "MEETING",
        start_date: "2026-10-25",
        end_date: "2026-10-25",
        requires_attendance: false,
    },
    {
        title: "Admission Open Day",
        event_type: "ADMISSION",
        start_date: "2026-11-02",
        end_date: "2026-11-02",
        requires_attendance: false,
    },
    {
        title: "Science Fair",
        event_type: "SPORTS",
        start_date: "2026-11-10",
        end_date: "2026-11-11",
        requires_attendance: true,
    },
];

function formatDate(iso: string) {
    const [y, m, d] = iso.split("-");
    const date = new Date(`${y}-${m}-${d}T00:00:00`);
    return date.toLocaleDateString("en-GB", { day: "2-digit", month: "short" });
}

export function UpcomingEvents() {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-sm">UPCOMING EVENTS · Next 14 days</CardTitle>
            </CardHeader>
            <CardContent>
                <ul className="space-y-3 text-sm">
                    {events.map((e) => {
                        const start = formatDate(e.start_date);
                        const end =
                            e.end_date !== e.start_date ? ` - ${formatDate(e.end_date)}` : "";
                        return (
                            <li key={e.title} className="flex items-center justify-between gap-3">
                                <div className="min-w-0">
                                    <div className="flex items-center gap-2">
                                        <span className="truncate font-medium">{e.title}</span>
                                        <span className="bg-muted rounded px-1.5 py-0.5 text-[10px] tracking-wide uppercase">
                                            {e.event_type}
                                        </span>
                                    </div>
                                    <div className="text-muted-foreground text-xs">
                                        {start}
                                        {end} {e.requires_attendance && "· ⚑ attendance"}
                                    </div>
                                </div>
                            </li>
                        );
                    })}
                </ul>
            </CardContent>
        </Card>
    );
}
