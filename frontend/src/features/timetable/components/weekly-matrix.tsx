import { useMemo } from "react";

const days = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

type Props = {
    rowCount: number;
};

export function WeeklyMatrix({ rowCount }: Props) {
    const rows = useMemo(() => Array.from({ length: rowCount }, (_, i) => i), [rowCount]);

    return (
        <div className="overflow-x-auto">
            <table className="w-full">
                <thead>
                    <tr>
                        {days.map((d) => (
                            <th key={d} className="border-t px-4 py-2 text-left font-medium">
                                {d}
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody>
                    {rows.map((i) => (
                        <tr key={i}>
                            {days.map((d) => (
                                <td key={d} className="text-muted-foreground px-4 py-16 align-top">
                                    {/* Future slot assignment cell */}
                                </td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
