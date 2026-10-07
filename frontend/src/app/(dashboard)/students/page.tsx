"use client";

import { StudentsTable } from "@/features/students";

export default function StudentsPage() {
    return (
        <div className="space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Students</h1>
            <StudentsTable />
        </div>
    );
}
