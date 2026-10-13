const todayItems = [
    "Attendance marked [18 of 24]",
    "Next: Grade 6 Simba · Maths [10:40 Room4]",
    "Event: Mid-term exams [Mon]",
];

export function TodayPanel() {
    return (
        <div className="bg-background rounded-sm border p-4">
            <h2 className="text-foreground mb-3 text-sm font-medium">TODAY · Tue 13 Oct</h2>
            <ul className="text-foreground space-y-2 text-sm">
                {todayItems.map((t) => (
                    <li key={t}>{t}</li>
                ))}
            </ul>
        </div>
    );
}
