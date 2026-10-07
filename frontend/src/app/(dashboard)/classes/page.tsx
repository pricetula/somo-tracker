"use client";

import { ClassesTable } from "@/features/classes";

export default function ClassesPage() {
    return (
        <div className="space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Classes</h1>
            <ClassesTable />
        </div>
    );
}
