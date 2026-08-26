import Link from "next/link";
import { Plus } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";
import { type Allocation } from "@/lib/api/timetable";

interface TimeSlotProps {
    isBreak: boolean;
    allocation: Allocation;
    dayOfWeek?: number;
    blockId?: string;
}

export function AllocationBlock({ allocation, isBreak, dayOfWeek, blockId }: TimeSlotProps) {
    if (!dayOfWeek || !blockId) return null;

    if (!allocation?.id) {
        return (
            <section className="flex items-center">
                <Link
                    href={`/timetable/allocate?block=${blockId}&day=${dayOfWeek}`}
                    className={buttonVariants({ variant: "outline" })}
                >
                    <Plus />
                    <span>Assign Teacher</span>
                </Link>
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
