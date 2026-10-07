"use client";

interface TeacherDetailProps {
    id: string;
}

export function TeacherDetail({ id }: TeacherDetailProps) {
    return (
        <div className="p-6">
            <h1 className="text-2xl font-semibold">Teacher</h1>
            <p className="mt-2">Teacher ID: {id}</p>
            <p className="text-muted-foreground mt-1">Name placeholder</p>
        </div>
    );
}
