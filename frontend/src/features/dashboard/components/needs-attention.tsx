const items = [
    { text: "Attendance not marked", meta: "6 classes", action: "Mark" },
    { text: "Outstanding fees", meta: "38 stud.", action: "View" },
    { text: "Classes without teacher", meta: "2 classes", action: "Asgn" },
    { text: "Pending guardian invites", meta: "17", action: "Remd" },
];

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function NeedsAttention() {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-sm">NEEDS ATTENTION (4 items)</CardTitle>
            </CardHeader>
            <CardContent>
                <ul className="space-y-2 text-sm">
                    {items.map((i) => (
                        <li key={i.text} className="flex items-center justify-between">
                            <span className="text-foreground">
                                {i.text} <span className="text-muted-foreground">[{i.meta}]</span>
                            </span>
                            <a href="#" className="text-primary text-xs">
                                [{i.action}]
                            </a>
                        </li>
                    ))}
                </ul>
            </CardContent>
        </Card>
    );
}
