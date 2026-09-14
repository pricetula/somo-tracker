"use client";

import { CurriculumAddForm } from "@/features/curriculum";

export default function CurriculumAddPage() {
    return (
        <div className="p-6">
            <h1 className="mb-4 text-2xl font-semibold">Add Subject</h1>
            <CurriculumAddForm />
        </div>
    );
}
