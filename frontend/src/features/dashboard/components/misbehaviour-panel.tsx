import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const counts = [
    { label: "Noise / disruption", value: 6, tag: "LOW · 2 OPEN" },
    { label: "Fighting / aggression", value: 2, tag: "HIGH · 1 RESOLVED" },
    { label: "Repeated flags", value: 3, tag: "MEDIUM · 3 STUDENTS" },
];

const incidents = [
    {
        time: "Today 08:40",
        student: "Kevin Mwangi",
        type: "NOISE",
        severity: "LOW",
        detail: "Classroom disruption — loud outbursts",
        status: "PARENT_NOTIFIED",
        action_taken: "Warning issued; parent contacted",
        witness_count: 1,
    },
    {
        time: "Yesterday 14:15",
        student: "Amina Hassan",
        type: "FIGHTING",
        severity: "HIGH",
        detail: "Push/shove during recess — minor scrape",
        status: "MEETING_SCHEDULED",
        action_taken: "Nurse check; meeting set Fri",
        witness_count: 3,
    },
    {
        time: "Yesterday 11:30",
        student: "Peter Otieno",
        type: "DISRUPTION",
        severity: "MEDIUM",
        detail: "Refused to follow instructions — repeated flag",
        status: "OPEN",
        action_taken: "Behaviour plan updated",
        witness_count: 0,
    },
];

function tagClass(t: string) {
    if (t === "NOISE") return "bg-rose-100 text-rose-700";
    if (t === "FIGHTING") return "bg-amber-100 text-amber-700";
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
                                <span
                                    className={`rounded px-1 text-[10px] font-medium ${i.severity === "HIGH" ? "bg-red-100 text-red-700" : i.severity === "MEDIUM" ? "bg-amber-100 text-amber-700" : "bg-emerald-100 text-emerald-700"}`}
                                >
                                    {i.severity}
                                </span>
                                <span className="text-muted-foreground truncate">{i.detail}</span>
                                <span className="text-primary ml-auto shrink-0 text-[10px]">
                                    {i.status}
                                </span>
                            </li>
                        ))}
                    </ul>
                </div>
            </CardContent>
        </Card>
    );
}
