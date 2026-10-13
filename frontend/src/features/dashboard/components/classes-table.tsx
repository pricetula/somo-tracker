import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";

const classes = [
    { class: "PP1", stream: "Simba", students: 18, teacher: "Mary Achieng", attendance: "17 / 18" },
    {
        class: "PP1",
        stream: "Red house",
        students: 17,
        teacher: "Joseph Kamau",
        attendance: "17 / 17",
    },
    {
        class: "PP2",
        stream: "Red house",
        students: 20,
        teacher: "Grace Wanjiru",
        attendance: "Not marked (Amber)",
    },
    {
        class: "Grade 1",
        stream: "Simba",
        students: 22,
        teacher: "Unassigned",
        attendance: "Not marked (Amber)",
    },
    {
        class: "Grade 2",
        stream: "Simba",
        students: 19,
        teacher: "Peter Otieno",
        attendance: "18 / 19",
    },
    {
        class: "Grade 2",
        stream: "Red house",
        students: 21,
        teacher: "Lucy Njeri",
        attendance: "21 / 21",
    },
];

export function ClassesTable() {
    return (
        <div className="space-y-3">
            <div className="flex items-center justify-between">
                <h2 className="text-foreground text-sm font-medium">CLASSES</h2>
                <div className="text-muted-foreground flex items-center gap-3 text-xs">
                    <span>All levels ▾</span>
                    <span>🔍</span>
                </div>
            </div>
            <div className="overflow-x-auto">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead className="text-muted-foreground">Class</TableHead>
                            <TableHead className="text-muted-foreground">Stream</TableHead>
                            <TableHead className="text-muted-foreground">Students</TableHead>
                            <TableHead className="text-muted-foreground">Class teacher</TableHead>
                            <TableHead className="text-muted-foreground">
                                Attendance today
                            </TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {classes.map((c, idx) => (
                            <TableRow key={idx}>
                                <TableCell>{c.class}</TableCell>
                                <TableCell>{c.stream}</TableCell>
                                <TableCell>{c.students}</TableCell>
                                <TableCell>{c.teacher}</TableCell>
                                <TableCell>{c.attendance}</TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>
            <p className="text-muted-foreground text-xs">Showing 6 of 28 View all classes -&gt;</p>
        </div>
    );
}
