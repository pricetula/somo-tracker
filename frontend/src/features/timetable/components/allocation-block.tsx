import Link from "next/link";
import { Plus, Trash } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { type Allocation } from "@/lib/api/timetable";

interface TimeSlotProps {
    isBreak: boolean;
    allocation: Allocation;
    dayOfWeek?: number;
    blockId?: string;
    classId?: string;
}

export function AllocationBlock({
    allocation,
    isBreak,
    dayOfWeek,
    blockId,
    classId,
}: TimeSlotProps) {
    if (!dayOfWeek || !blockId) return null;

    if (!allocation?.id) {
        return (
            <Link
                href={`/timetable/allocate?block=${blockId}&day=${dayOfWeek}&class=${classId ?? ""}`}
                className={buttonVariants({ variant: "outline" })}
            >
                <Plus />
                <span>Assign Teacher</span>
            </Link>
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
        <section className="space-y-2">
            <header className="flex items-center justify-between">
                <h4>
                    {allocation.learning_area_id && allocation.learning_area_name && (
                        <Link
                            href={`/curriculum/${allocation.learning_area_id}`}
                            className="truncate"
                        >
                            {allocation.learning_area_name}
                        </Link>
                    )}
                </h4>
                <Button size="xs" variant="outline">
                    <Trash />
                </Button>
            </header>

            {allocation.teacher_name && allocation.teacher_id && (
                <Link href={`/teachers/${allocation.teacher_id}`} className="truncate">
                    {allocation.teacher_name}
                </Link>
            )}

            {allocation.room_identifier && (
                <p className="text-muted-foreground truncate">{allocation.room_identifier}</p>
            )}
        </section>
    );
}
