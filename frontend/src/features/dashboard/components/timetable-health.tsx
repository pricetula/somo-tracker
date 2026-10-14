import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const measures = [
    { label: "Lesson slots filled", value: 82, total: 100, unit: "%" },
    { label: "Classes with no lessons", value: 2, total: 28, unit: "" },
    { label: "Teachers with no lessons", value: 3, total: 24, unit: "" },
];

export function TimetableHealth() {
    return (
        <Card className="h-full">
            <CardHeader className="pb-2">
                <CardTitle className="text-sm">Timetable health · Term 3</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
                {measures.map((m) => (
                    <div key={m.label}>
                        <div className="mb-1 flex items-baseline justify-between text-sm">
                            <span className="text-muted-foreground">{m.label}</span>
                            <span className="font-medium">
                                {m.value}
                                {m.unit}
                            </span>
                        </div>
                        <div className="bg-muted flex h-1.5 w-full overflow-hidden rounded-full">
                            <div
                                className="h-full rounded-full bg-emerald-600"
                                style={{ width: `${(m.value / (m.total || 100)) * 100}%` }}
                            />
                        </div>
                        {m.unit === "" && (
                            <div className="text-muted-foreground mt-0.5 text-xs">of {m.total}</div>
                        )}
                    </div>
                ))}
            </CardContent>
        </Card>
    );
}
