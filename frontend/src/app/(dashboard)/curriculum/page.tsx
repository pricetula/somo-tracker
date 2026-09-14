"use client";

import { CurriculumTable } from "@/features/curriculum";

export default function CurriculumPage() {
    return (
        <div className="space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Subjects</h1>
            <CurriculumTable />
        </div>
    );
}
