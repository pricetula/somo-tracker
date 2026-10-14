import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type CoverageRow = {
    grade: string;
    students: string;
    teachers: string;
    marked: string;
};

const rows: CoverageRow[] = [
    { grade: "PP1", students: "18/18", teachers: "2/2", marked: "100%" },
    { grade: "Grade 1", students: "22/22", teachers: "1/2", marked: "50%" },
    { grade: "Grade 2", students: "40/40", teachers: "2/2", marked: "100%" },
];

export function ClassCoverageMini() {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-sm">CLASS COVERAGE</CardTitle>
            </CardHeader>
            <CardContent>
                <ul className="space-y-2 text-sm">
                    {rows.map((r) => {
                        const markedOk = r.marked === "100%";
                        return (
                            <li key={r.grade} className="flex items-center justify-between">
                                <span className="font-medium">{r.grade}</span>
                                <span className="text-muted-foreground">
                                    {r.students} · {r.teachers} teachers
                                </span>
                                <span className={markedOk ? "text-emerald-600" : "text-amber-600"}>
                                    {r.marked} marked
                                </span>
                            </li>
                        );
                    })}
                </ul>
            </CardContent>
        </Card>
    );
}
