import { Card, CardContent } from "@/components/ui/card";
import type { EnrollmentData } from "./types";

function StatChip({
    label,
    count,
    colorClass,
    showDash,
}: {
    label: string;
    count: number;
    colorClass: string;
    showDash?: boolean;
}) {
    return (
        <span className="flex flex-col items-center">
            <span className="text-muted-foreground text-[11px]">{label}</span>
            <span className={`text-[20px] font-bold ${colorClass}`}>
                {showDash && count === 0 ? "—" : count}
            </span>
        </span>
    );
}

export function EnrollmentMetrics({ data }: { data: EnrollmentData }) {
    return (
        <Card>
            <CardContent>
                <div className="flex items-center justify-around">
                    <StatChip label="Active" count={data.active} colorClass="text-emerald-600" />
                    <span className="border-border h-8 border-r" />
                    <StatChip
                        label="Suspended"
                        count={data.suspended}
                        colorClass="text-amber-600"
                    />
                    <span className="border-border h-8 border-r" />
                    <StatChip
                        label="Transferred"
                        count={data.transferred}
                        colorClass="text-muted-foreground"
                        showDash
                    />
                </div>
            </CardContent>
        </Card>
    );
}
