"use client";

import { TimetableHeader, PeriodSidebar } from "./TimetableHeader";
import { TimetableCell } from "./TimetableCell";
import type { TimetableGridView } from "../types";

interface TimetableGridProps {
    gridView: TimetableGridView;
    onCellAdd?: (structureId: string) => void;
    onCellEdit?: (slotId: string) => void;
    onCellDelete?: (slotId: string) => void;
    isReadOnly?: boolean;
}

export function TimetableGrid({
    gridView,
    onCellAdd,
    onCellEdit,
    onCellDelete,
    isReadOnly = false,
}: TimetableGridProps) {
    const { periods, days } = gridView;

    return (
        <div className="overflow-x-auto">
            <table className="w-full min-w-[1000px] border-collapse">
                <TimetableHeader periods={periods} />
                <PeriodSidebar periods={periods} />
                <tbody>
                    {periods.map((period, periodIndex) => (
                        <tr key={period.id}>
                            {days.map((day) => (
                                <td
                                    key={`${period.id}-${day.day}`}
                                    className="border-border relative border"
                                >
                                    <TimetableCell
                                        cell={day.cells[periodIndex]}
                                        onAdd={onCellAdd}
                                        onEdit={onCellEdit}
                                        onDelete={onCellDelete}
                                        isReadOnly={isReadOnly}
                                    />
                                </td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
