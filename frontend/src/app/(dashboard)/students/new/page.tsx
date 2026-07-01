/**
 * Create Student page.
 *
 * Maps to POST /api/v1/students.
 * On success, navigates to the student detail page.
 */

"use client";

import { useRouter } from "next/navigation";

import { StudentForm } from "@/features/students";

export default function NewStudentPage() {
    const router = useRouter();

    const handleSuccess = (id: string) => {
        router.push(`/students/${id}`);
    };

    return (
        <div className="mx-auto flex max-w-xl flex-col px-6 pt-6 pb-8">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">Add New Student</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Enter the student&apos;s demographic information. You can enroll them in a class
                    after creating their profile.
                </p>
            </div>

            <StudentForm mode="create" onSuccess={handleSuccess} />
        </div>
    );
}
