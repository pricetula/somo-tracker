import { Button } from "@/components/ui/button";
import { TimeSlotRow } from "./time-slot-row";
import type { TimeSlotDraft } from "../types/timetable-template";
import { formatDateString } from "@/lib/utils/date";

type Props = {
    slots: TimeSlotDraft[];
    editable?: boolean;
    onSlotChange?: (id: string, patch: Partial<TimeSlotDraft>) => void;
    onSlotDelete?: (id: string) => void;
    onAddSlot?: () => void;
    days?: string[];
};

const DEFAULT_DAYS = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

export function TimetableGrid({
    slots,
    editable = false,
    onSlotChange,
    onSlotDelete,
    onAddSlot,
    days = DEFAULT_DAYS,
}: Props) {
    return (
        <div className="space-y-4">
            <div className="overflow-x-auto rounded-md border">
                <table className="w-full">
                    <thead>
                        <tr className="border-b">
                            <th className="bg-background text-muted-foreground sticky top-0 left-0 z-30 w-96 border-r px-4 py-3 text-left font-medium">
                                Time Slot
                            </th>
                            {days.map((d) => (
                                <th
                                    key={d}
                                    className="bg-background sticky top-0 z-20 min-w-56 border-r px-4 py-3 text-left font-medium"
                                >
                                    {d}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {slots.map((slot) => (
                            <tr key={slot.id} className="border-b align-top">
                                <td className="bg-background sticky left-0 z-10 border-r">
                                    {editable && onSlotChange && onSlotDelete ? (
                                        <TimeSlotRow
                                            slot={slot}
                                            onChange={(patch) => onSlotChange(slot.id, patch)}
                                            onDelete={() => onSlotDelete(slot.id)}
                                            canDelete={slots.length > 1}
                                        />
                                    ) : (
                                        <div className="min-h-31 w-48 space-y-1 p-3 pt-4">
                                            <div className="mb-4 font-medium">{slot.name}</div>
                                            <div className="text-muted-foreground text-xs">
                                                {`${formatDateString(slot.start_time, {
                                                    inputFormat: "HH:mm",
                                                    outputFormat: "HH:mm a",
                                                })} — ${formatDateString(slot.end_time, {
                                                    inputFormat: "HH:mm",
                                                    outputFormat: "HH:mm a",
                                                })}`}
                                            </div>
                                        </div>
                                    )}
                                </td>
                                {days.map((d) =>
                                    !slot.is_instructional ? (
                                        <td
                                            key={d}
                                            className="bg-row-disabled border-r p-4 text-center align-middle"
                                        >
                                            {slot.name}
                                        </td>
                                    ) : (
                                        <td key={d} className="border-r px-4 py-4 align-top" />
                                    )
                                )}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>

            {editable && onAddSlot && (
                <div className="flex items-center justify-between">
                    <div>
                        <Button type="button" variant="outline" onClick={onAddSlot}>
                            + Add Time Slot
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
}
