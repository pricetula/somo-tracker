"use client";

import { ClassAddForm } from "@/features/classes";

export default function ClassAddPage() {
    return (
        <div className="p-6">
            <h1 className="mb-4 text-2xl font-semibold">Add Class</h1>
            <ClassAddForm />
        </div>
    );
}
