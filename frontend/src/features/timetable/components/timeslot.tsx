import type { EnrichedSlot } from "@/lib/api/timetable-structure";

interface TimeSlotProps {
    slot?: EnrichedSlot | null;
    isBreak?: boolean;
}

export function TimeSlot({ slot, isBreak = false }: TimeSlotProps) {
    if (isBreak) {
        return (
            <section className="class-info break-slot">
                <span className="text-muted-foreground text-xs italic">Break</span>
            </section>
        );
    }

    if (!slot) {
        return (
            <section className="class-info empty-slot">
                <span className="text-muted-foreground text-xs">—</span>
            </section>
        );
    }

    return (
        <section className="class-info filled-slot">
            <h4 className="truncate text-xs font-medium">{slot.learning_area_name ?? "TBA"}</h4>
            <p className="text-muted-foreground truncate text-[10px]">
                {slot.teacher_name ? <strong>{slot.teacher_name}</strong> : "TBA"}
            </p>
            <p className="text-muted-foreground truncate text-[10px]">
                {slot.room_identifier ? <strong>{slot.room_identifier}</strong> : "TBA"}
            </p>
        </section>
    );
}
