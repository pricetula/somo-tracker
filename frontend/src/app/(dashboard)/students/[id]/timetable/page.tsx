"use client";

import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

type StudentTimetableSlot = {
    day_of_week: number;
    time_slot_name: string;
    start_time: string;
    end_time: string;
    subject_name: string;
    teacher_name: string;
    room_name: string;
    class_name: string;
    grade_name: string;
};

export default function StudentTimetablePage() {
    const params = useParams<{ id: string }>();
    const { data, isLoading } = useQuery<StudentTimetableSlot[]>({
        queryKey: ["student-timetable", params.id],
        queryFn: async () => {
            const res = await api.get<StudentTimetableSlot[]>(
                `/api/students/${params.id}/timetable`
            );
            return res;
        },
        enabled: !!params.id,
    });

    if (isLoading) {
        return (
            <div className="space-y-4 p-6">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-40 w-full" />
            </div>
        );
    }

    return (
        <div className="space-y-6 p-6">
            <h1 className="text-2xl font-semibold">Student Timetable</h1>
            {data && data.length > 0 ? (
                <div className="grid gap-4">
                    {Array.from(new Set(data.map((d) => d.day_of_week))).map((day) => (
                        <Card key={day}>
                            <CardHeader>
                                <CardTitle>Day {day}</CardTitle>
                            </CardHeader>
                            <CardContent className="space-y-2">
                                {data
                                    .filter((d) => d.day_of_week === day)
                                    .map((slot, i) => (
                                        <div key={i} className="rounded border p-3">
                                            <div className="font-medium">
                                                {slot.time_slot_name} {slot.start_time}-
                                                {slot.end_time}
                                            </div>
                                            <div>
                                                {slot.subject_name} • {slot.teacher_name} •{" "}
                                                {slot.room_name} • {slot.class_name}
                                            </div>
                                        </div>
                                    ))}
                            </CardContent>
                        </Card>
                    ))}
                </div>
            ) : (
                <Card>
                    <CardContent className="text-muted-foreground p-6">
                        No timetable data for this student.
                    </CardContent>
                </Card>
            )}
        </div>
    );
}
