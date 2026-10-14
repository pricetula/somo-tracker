export function KpiStrip() {
    const items = [
        { label: "Attendance today", value: "94%", sub: "↑ 2% vs yesterday" },
        { label: "Fees collected", value: "68%", sub: "38 students outstanding" },
    ];

    return (
        <div className="mb-6 grid grid-cols-1 gap-3 md:grid-cols-2">
            {items.map((i) => (
                <div key={i.label} className="flex items-center justify-between border p-3">
                    <div>
                        <div className="text-muted-foreground text-sm">{i.label}</div>
                        <div className="text-2xl">{i.value}</div>
                    </div>
                    <div className="text-muted-foreground text-right text-xs">{i.sub}</div>
                </div>
            ))}
        </div>
    );
}
