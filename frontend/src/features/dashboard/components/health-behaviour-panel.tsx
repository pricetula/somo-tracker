import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const stats = [
    { label: "Nurse visits today", value: "3", sub: "2 ailments · 1 first-aid" },
    { label: "Behaviour flags", value: "4", sub: "2 positive · 2 follow-up" },
    { label: "Vaccination overdue", value: "12", sub: "Grade 1–3 · due Mon" },
];

const notes = [
    {
        time: "09:15",
        student: "Amara Kimani",
        type: "Health",
        detail: "Fever — sent to nurse, parent called",
    },
    {
        time: "10:40",
        student: "David Ochieng",
        type: "Behaviour",
        detail: "Positive leadership — class assembly",
    },
    {
        time: "11:05",
        student: "Grace Wanjiru",
        type: "Health",
        detail: "Allergy — medication given, form signed",
    },
];

export function HealthBehaviourPanel() {
    return (
        <Card>
            <CardHeader className="pb-2">
                <CardTitle className="text-sm">Health & behaviour</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
                <div className="grid grid-cols-3 gap-3">
                    {stats.map((s) => (
                        <div key={s.label} className="rounded border px-3 py-3">
                            <div className="text-muted-foreground text-xs">{s.label}</div>
                            <div className="mt-0.5 text-xl font-medium">{s.value}</div>
                            <div className="text-muted-foreground text-[11px]">{s.sub}</div>
                        </div>
                    ))}
                </div>
                <div className="bg-muted/40 rounded px-3 py-3">
                    <div className="mb-1 text-xs font-medium">Recent notes</div>
                    <ul className="space-y-2 text-xs">
                        {notes.map((n) => (
                            <li key={n.time + n.student} className="flex gap-2">
                                <span className="text-muted-foreground shrink-0">{n.time}</span>
                                <span className="font-medium">{n.student}</span>
                                <span
                                    className={`rounded px-1 text-[10px] ${n.type === "Health" ? "bg-sky-100 text-sky-700" : "bg-violet-100 text-violet-700"}`}
                                >
                                    {n.type}
                                </span>
                                <span className="text-muted-foreground truncate">{n.detail}</span>
                            </li>
                        ))}
                    </ul>
                </div>
            </CardContent>
        </Card>
    );
}
