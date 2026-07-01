/**
 * Create Parent page.
 *
 * Maps to POST /api/v1/parents.
 * On success, navigates to the parent detail page.
 */

"use client";

import { useRouter } from "next/navigation";

import { CreateParentForm } from "@/features/parents";

export default function NewParentPage() {
    const router = useRouter();

    const handleSuccess = (id: string) => {
        router.push(`/parents/${id}`);
    };

    return (
        <div className="mx-auto flex max-w-xl flex-col px-6 pt-6 pb-8">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">Add Parent / Guardian</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Create a parent profile to manage guardian communication, M-Pesa billing
                    notifications, and SMS alerts. The user must already exist in the system.
                </p>
            </div>

            <CreateParentForm onSuccess={handleSuccess} />
        </div>
    );
}
