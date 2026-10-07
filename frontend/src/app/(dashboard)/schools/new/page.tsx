"use client";

/**
 * Full-page create school route.
 */

import { CreateSchoolForm } from "@/features/school";

export default function NewSchoolPage() {
    return (
        <div className="mx-auto max-w-xl space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Create School</h1>
            <CreateSchoolForm />
        </div>
    );
}
