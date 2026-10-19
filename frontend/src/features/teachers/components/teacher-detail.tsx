"use client";

import { useQuery } from "@tanstack/react-query";
import { getTeacher } from "@/lib/api/teachers";

interface TeacherDetailProps {
    id: string;
}

export function TeacherDetail({ id }: TeacherDetailProps) {
    const { data, isLoading } = useQuery({
        queryKey: ["teacher", id],
        queryFn: () => getTeacher(id),
    });

    if (isLoading) return <div className="p-6">Loading...</div>;
    if (!data) return <div className="p-6">Teacher not found</div>;

    return (
        <div className="space-y-6 p-6">
            <div className="space-y-1">
                <h1 className="text-2xl font-semibold">{data.full_name}</h1>
                <p className="text-muted-foreground">{data.email}</p>
            </div>
            <div className="space-y-2 text-sm">
                <div>
                    <span className="font-medium">Status:</span>{" "}
                    {data.is_active ? "Active" : "Inactive"}
                </div>
                <div>
                    <span className="font-medium">Created:</span>{" "}
                    {new Date(data.created_at).toLocaleDateString()}
                </div>
            </div>
            <div className="space-y-4">
                <h2 className="text-lg font-semibold">Classes Taught</h2>
                <div className="text-muted-foreground text-sm">
                    Classes taught will be listed here once enriched endpoint is available.
                </div>
            </div>
            <div className="space-y-4">
                <h2 className="text-lg font-semibold">Timetable Summary</h2>
                <div className="text-muted-foreground text-sm">
                    Timetable summary per term will be displayed here.
                </div>
            </div>
        </div>
    );
}
