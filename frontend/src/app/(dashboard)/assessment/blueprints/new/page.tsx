/**
 * Create Blueprint page.
 *
 * Maps to POST /api/v1/assessment/blueprints.
 * On success, navigates to the blueprint detail page for indicator linking.
 */

"use client";

import { useRouter } from "next/navigation";

import { BlueprintForm } from "@/features/assessment";

export default function NewBlueprintPage() {
    const router = useRouter();

    const handleSuccess = (id: string) => {
        router.push(`/assessment/blueprints/${id}`);
    };

    return (
        <div className="mx-auto flex max-w-xl flex-col px-6 pt-6 pb-8">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">New Assessment Blueprint</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Define an assessment blueprint by selecting a grade level, assessment type, and
                    term. After creating, you can link performance indicators from the curriculum.
                </p>
            </div>

            <BlueprintForm onSuccess={handleSuccess} />
        </div>
    );
}
