import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const stats = [
    { label: "Nurse visits today", value: "3", sub: "2 AILMENT · 1 FIRST_AID" },
    { label: "Behaviour flags", value: "4", sub: "2 POSITIVE · 2 OPEN" },
    { label: "Vaccination overdue", value: "12", sub: "MMR / Polio · OVERDUE" },
];

const notes = [
    {
        time: "09:15",
        student: "Amara Kimani",
        type: "AILMENT",
        detail: "Fever — ACTIVE / parent notified",
        status: "ACTIVE",
        recorded_by: "Nurse J. Otieno",
    },
    {
        time: "10:40",
        student: "David Ochieng",
        type: "CHECKUP",
        detail: "Routine vision — RESOLVED",
        status: "RESOLVED",
        recorded_by: "Nurse J. Otieno",
    },
    {
        time: "11:05",
        student: "Grace Wanjiru",
        type: "FIRST_AID",
        detail: "Allergy — medication given · ACTIVE",
        status: "ACTIVE",
        recorded_by: "Nurse J. Otieno",
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
                                    className={`rounded px-1 text-[10px] font-medium ${
                                        n.type === "AILMENT"
                                            ? "bg-rose-100 text-rose-700"
                                            : n.type === "FIRST_AID"
                                              ? "bg-sky-100 text-sky-700"
                                              : n.type === "CHECKUP"
                                                ? "bg-violet-100 text-violet-700"
                                                : "bg-muted text-muted-foreground"
                                    }`}
                                >
                                    {n.type}
                                </span>
                                <span className="text-muted-foreground truncate">{n.detail}</span>
                                <span className="text-muted-foreground ml-auto shrink-0 text-[10px]">
                                    {(n as { status: string; recorded_by: string }).status} ·{" "}
                                    {(n as { status: string; recorded_by: string }).recorded_by}
                                </span>
                            </li>
                        ))}
                    </ul>
                </div>
            </CardContent>
        </Card>
    );
}
