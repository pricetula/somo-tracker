"use client";

interface MomentumArrowListProps {
    data: MomentumData[];
}
export interface MomentumData {
    momentumScore: number;
    subjectName?: string;
}

import { MomentumArrow } from "./momentum-arrow";

export function MomentumArrowList({ data }: MomentumArrowListProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-4 text-center text-sm">
                No momentum data available.
            </p>
        );
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">Subject Momentum</p>
            <div className="space-y-1">
                {data.map((item) => (
                    <MomentumArrow key={item.subjectName} data={item} />
                ))}
            </div>
        </div>
    );
}
