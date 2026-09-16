import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Trash2 } from "lucide-react";
import type { TimeSlotDraft } from "../types/timetable-template";

type Props = {
    slot: TimeSlotDraft;
    onChange: (patch: Partial<TimeSlotDraft>) => void;
    onDelete: () => void;
    canDelete: boolean;
};

export function TimeSlotRow({ slot, onChange, onDelete, canDelete }: Props) {
    return (
        <div className="space-y-2 p-3">
            <div>
                <Input
                    value={slot.name}
                    onChange={(e) => onChange({ name: e.target.value })}
                    placeholder="Slot name"
                />
            </div>
            <div className="flex gap-2">
                <Input
                    type="time"
                    value={slot.start_time}
                    onChange={(e) => onChange({ start_time: e.target.value })}
                />
                <Input
                    type="time"
                    value={slot.end_time}
                    onChange={(e) => onChange({ end_time: e.target.value })}
                />
            </div>
            <div className="flex items-center justify-between">
                <label className="flex items-center gap-2">
                    <Checkbox
                        checked={slot.is_instructional}
                        onCheckedChange={(checked) => onChange({ is_instructional: !!checked })}
                    />
                    Instructional
                </label>
                <Button
                    variant="ghost"
                    size="icon"
                    onClick={onDelete}
                    disabled={!canDelete}
                    aria-label="Delete time slot"
                >
                    <Trash2 className="h-4 w-4" />
                </Button>
            </div>
        </div>
    );
}
