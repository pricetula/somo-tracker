import { type Allocation } from "@/lib/api/timetable";

interface TimeSlotProps {
    isBreak: boolean;
    allocation: Allocation;
}

export function AllocationBlock({ allocation, isBreak }: TimeSlotProps) {
    if (!allocation.id) {
        return (
            <section className="class-info empty-slot">
                <span className="text-muted-foreground text-xs">—</span>
            </section>
        );
    }

    if (isBreak) {
        return (
            <section className="class-info break-slot p-4">
                <span className="text-muted-foreground text-xs italic">
                    {allocation.learning_area_name ?? "Break"}
                </span>
            </section>
        );
    }

    return (
        <section className="class-info filled-slot p-4">
            <h4 className="truncate text-xs font-medium">
                {allocation.learning_area_name ?? "TBA"}
            </h4>
            <p className="text-muted-foreground truncate text-[10px]">
                {allocation.teacher_name ? <strong>{allocation.teacher_name}</strong> : "TBA"}
            </p>
            <p className="text-muted-foreground truncate text-[10px]">
                {allocation.room_identifier ? <strong>{allocation.room_identifier}</strong> : "TBA"}
            </p>
        </section>
    );
}
