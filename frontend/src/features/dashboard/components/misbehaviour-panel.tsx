import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const counts = [
    { label: "Noise / disruption", value: 6, tag: "today" },
    { label: "Fighting / aggression", value: 2, tag: "this week" },
    { label: "Repeated flags", value: 3, tag: "3 students" },
];

const incidents = [
    {
        time: "Today 08:40",
        student: "Kevin Mwangi",
        type: "Noise",
        detail: "Classroom disruption — loud outbursts",
        status: "Parent notified",
    },
    {
        time: "Yesterday 14:15",
        student: "Amina Hassan",
        type: "Fighting",
        detail: "Push/shove during recess — minor scrape",
        status: "Meeting set Fri",
    },
    {
        time: "Yesterday 11:30",
        student: "Peter Otieno",
        type: "Disruption",
        detail: "Refused to follow instructions — repeated",
        status: "Follow-up",
    },
];

function tagClass(t: string) {
    if (t === "Noise") return "bg-rose-100 text-rose-700";
    if (t === "Fighting") return "bg-amber-100 text-amber-700";
    return "bg-violet-100 text-violet-700";
}

export function MisbehaviourPanel() {
    return (
        <Card>
            <CardHeader className="pb-2">
                <CardTitle className="text-sm">Student misbehaviour</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
                <div className="grid grid-cols-3 gap-3">
                    {counts.map((c) => (
                        <div key={c.label} className="rounded border px-3 py-3">
                            <div className="text-muted-foreground text-xs">{c.label}</div>
                            <div className="mt-0.5 text-xl font-medium">{c.value}</div>
                            <div className="text-muted-foreground text-[11px]">{c.tag}</div>
                        </div>
                    ))}
                </div>
                <div className="bg-muted/40 rounded px-3 py-3">
                    <div className="mb-1.5 text-xs font-medium">Recent incidents</div>
                    <ul className="space-y-2 text-xs">
                        {incidents.map((i) => (
                            <li key={i.time + i.student} className="flex items-start gap-2">
                                <span className="text-muted-foreground shrink-0">{i.time}</span>
                                <span className="shrink-0 font-medium">{i.student}</span>
                                <span
                                    className={`rounded px-1 text-[10px] font-medium ${tagClass(i.type)}`}
                                >
                                    {i.type}
                                </span>
                                <span className="text-muted-foreground truncate">{i.detail}</span>
                                <span className="text-primary ml-auto shrink-0">{i.status}</span>
                            </li>
                        ))}
                    </ul>
                </div>
            </CardContent>
        </Card>
    );
}
