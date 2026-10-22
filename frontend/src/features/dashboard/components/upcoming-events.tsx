import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import Link from "next/link";

type EventRow = {
    date: string; // "Mon 19"
    title: string;
    typeTag: string; // Exam, Sports, Meeting, Admission, Holiday
    isPast?: boolean;
    requiresAttendance?: boolean;
    hasMarkedAttendance?: boolean; // if false and requiresAttendance and past => pin with Mark
};

const events: EventRow[] = [
    // Pinned past event with unmarked attendance
    {
        date: "Fri 9",
        title: "Career day",
        typeTag: "Meeting",
        isPast: true,
        requiresAttendance: true,
        hasMarkedAttendance: false,
    },
    // Upcoming / current
    { date: "Mon 19", title: "Mid-term exams", typeTag: "Exam", requiresAttendance: true },
    { date: "Tue 20", title: "Mashujaa Day", typeTag: "Holiday" },
    { date: "Fri 23", title: "Parents' meeting", typeTag: "Meeting" },
    { date: "Sat 31", title: "Sports day", typeTag: "Sports" },
];

function tagColor(tag: string) {
    if (tag === "Holiday") return "bg-muted text-muted-foreground";
    if (tag === "Exam") return "bg-rose-100 text-rose-700";
    if (tag === "Sports") return "bg-sky-100 text-sky-700";
    if (tag === "Meeting") return "bg-violet-100 text-violet-700";
    if (tag === "Admission") return "bg-amber-100 text-amber-700";
    return "bg-muted text-muted-foreground";
}

export function UpcomingEvents() {
    return (
        <Card className="h-full">
            <CardHeader className="pb-2">
                <CardTitle className="text-sm">Events · Next 30 days</CardTitle>
            </CardHeader>
            <CardContent>
                <ul className="space-y-3 text-sm">
                    {events.map((e) => (
                        <li
                            key={e.title + e.date}
                            className="flex items-start justify-between gap-3"
                        >
                            <div className="min-w-0 flex-1">
                                <div className="flex items-center gap-2">
                                    <span
                                        className={`rounded px-1.5 py-0.5 text-[10px] font-medium tracking-wide uppercase ${tagColor(e.typeTag)}`}
                                    >
                                        {e.typeTag}
                                    </span>
                                    <span
                                        className={`truncate font-medium ${e.typeTag === "Holiday" ? "text-muted-foreground" : "text-foreground"}`}
                                    >
                                        {e.title}
                                    </span>
                                </div>
                                <div className="text-muted-foreground mt-0.5 text-xs">{e.date}</div>
                            </div>
                            {e.isPast && e.requiresAttendance && !e.hasMarkedAttendance && (
                                <a
                                    href="#"
                                    className="text-primary mt-0.5 shrink-0 text-xs font-medium underline underline-offset-2"
                                >
                                    [Mark]
                                </a>
                            )}
                        </li>
                    ))}
                </ul>
                <div className="text-muted-foreground mt-4 text-xs">
                    <Link href="/events" className="underline">
                        View all events →
                    </Link>
                </div>
            </CardContent>
        </Card>
    );
}
