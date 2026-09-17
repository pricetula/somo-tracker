"use client";

import { useRouter, useParams, useSearchParams } from "next/navigation";
import { useEffect } from "react";
import { AssignSlotForm } from "@/features/timetable/components/assign-slot-form";

export default function AssignSlotPage() {
    const router = useRouter();
    const params = useParams();
    const searchParams = useSearchParams();

    const id = params.id as string;
    const dayParam = searchParams.get("day");
    const slotParam = searchParams.get("slot");

    const dayOfWeek = Number(dayParam ?? 1);
    const timeSlotId = slotParam ?? "";

    useEffect(() => {
        if (!timeSlotId) {
            router.replace(`/timetable/${id}`);
        }
    }, [timeSlotId, id, router]);

    if (!timeSlotId) {
        return null;
    }

    return (
        <div className="mx-auto max-w-2xl space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Assign Timetable Slot</h1>
            <AssignSlotForm
                dayOfWeek={dayOfWeek}
                timeSlotId={timeSlotId}
                onSuccess={() => router.push(`/timetable/${id}`)}
            />
        </div>
    );
}
