const items = [
    { text: "Attendance not marked", meta: "6 classes", action: "Mark" },
    { text: "Outstanding fees", meta: "38 stud.", action: "View" },
    { text: "Classes without teacher", meta: "2 classes", action: "Asgn" },
    { text: "Pending guardian invites", meta: "17", action: "Remd" },
];

export function NeedsAttention() {
    return (
        <div className="bg-background rounded-sm border p-4">
            <h2 className="text-foreground mb-3 text-sm font-medium">NEEDS ATTENTION (4 items)</h2>
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
        </div>
    );
}
