"use client";

interface StudentDetailProps {
    id: string;
}

export function StudentDetail({ id }: StudentDetailProps) {
    return (
        <div className="p-6">
            <h1 className="text-2xl font-semibold">Student</h1>
            <p className="mt-2">Student ID: {id}</p>
            <p className="text-muted-foreground mt-1">Name placeholder</p>
        </div>
    );
}
