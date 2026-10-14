const todayItems = [
    "Attendance marked [18 of 24]",
    "Next: Grade 6 Simba · Maths [10:40 Room4]",
    "Event: Mid-term exams [Mon]",
];

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function TodayPanel() {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-sm">TODAY · Tue 13 Oct</CardTitle>
            </CardHeader>
            <CardContent>
                <ul className="text-foreground space-y-2 text-sm">
                    {todayItems.map((t) => (
                        <li key={t}>{t}</li>
                    ))}
                </ul>
            </CardContent>
        </Card>
    );
}
