const summaries = [
    { label: "STUDENTS", value: "412", sub: "52% boys · 48% girls" },
    { label: "TEACHERS", value: "24", sub: "2 invites pending" },
    { label: "GUARDIANS", value: "389", sub: "17 invites pending" },
    { label: "FINANCE", value: "3", sub: "" },
    { label: "ADMINS", value: "2", sub: "" },
];

export function SummaryCards() {
    return (
        <div className="grid grid-cols-2 pb-4 md:grid-cols-3 lg:grid-cols-5">
            {summaries.map((s) => (
                <div key={s.label} className="bg-background border border-r-0 p-3">
                    <p className="text-muted-foreground font-medium">{s.label}</p>
                    <p className="text-foreground mt-1 text-2xl font-medium">{s.value}</p>
                    {s.sub && <p className="text-muted-foreground mt-1">{s.sub}</p>}
                </div>
            ))}
        </div>
    );
}
