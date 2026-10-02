"use client";

import { SubstitutionsTable } from "@/features/timetable";

export default function SubstitutionsPage() {
    return (
        <div className="space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Timetable Substitutions</h1>
            <SubstitutionsTable />
        </div>
    );
}
