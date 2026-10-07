"use client";

import { TeachersTable } from "@/features/teachers";

export default function TeachersPage() {
    return (
        <div className="space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Teachers</h1>
            <TeachersTable />
        </div>
    );
}
