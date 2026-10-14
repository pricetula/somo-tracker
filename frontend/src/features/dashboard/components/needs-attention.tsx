import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// Status dot: warning (today) vs muted (soon)
function Dot({ urgent }: { urgent?: boolean }) {
    return (
        <span
            className={`inline-block h-2 w-2 rounded-full ${urgent ? "bg-amber-500" : "bg-muted-foreground"}`}
            aria-label={urgent ? "do today" : "do soon"}
        />
    );
}

const items = [
    { label: "Attendance not marked", meta: "6 classes", action: "Mark", urgent: true },
    { label: "Students without a guardian", meta: "12", action: "Link", urgent: true },
    { label: "Classes with no lessons", meta: "2", action: "Timetable", urgent: false },
    { label: "Students not in a class", meta: "5", action: "Enroll", urgent: false },
];

export function NeedsAttention() {
    const hasItems = items.length > 0;
    if (!hasItems) {
        return (
            <Card>
                <CardContent className="text-muted-foreground py-4 text-sm">
                    Nothing needs attention
                </CardContent>
            </Card>
        );
    }
    return (
        <Card>
            <CardHeader className="pb-2">
                <CardTitle className="text-sm">Needs attention</CardTitle>
            </CardHeader>
            <CardContent>
                <ul className="space-y-3 text-sm">
                    {items.map((i) => (
                        <li key={i.label} className="flex items-center gap-3">
                            <Dot urgent={i.urgent} />
                            <div className="min-w-0 flex-1">
                                <div className="truncate">{i.label}</div>
                                <div className="text-muted-foreground text-xs">{i.meta}</div>
                            </div>
                            <a
                                href="#"
                                className="text-primary shrink-0 text-xs font-medium underline underline-offset-2"
                            >
                                [{i.action}]
                            </a>
                        </li>
                    ))}
                </ul>
            </CardContent>
        </Card>
    );
}
