/**
 * Enrollment Page — Full page render for /classes/:id/enroll.
 *
 * Academic year and term are resolved server-side from the current active term.
 * No user selection is required.
 */

"use client";

import { Suspense, use } from "react";
import { useRouter } from "next/navigation";

import { EnrollStudentsPanel } from "@/features/classes";

interface Props {
    params: Promise<{ id: string }>;
}

function EnrollStudentsContent({ classId }: { classId: string }) {
    const router = useRouter();

    return (
        <div className="p-6">
            <div className="mx-auto max-w-lg">
                <h1 className="mb-1 text-lg font-semibold">Enroll Students</h1>
                <p className="text-muted-foreground mb-6">
                    Search and select students to enroll in this class. The current academic term
                    will be used automatically.
                </p>
                <EnrollStudentsPanel classId={classId} onSuccess={() => router.back()} />
            </div>
        </div>
    );
}

export default function EnrollStudentsPage({ params }: Props) {
    const { id } = use(params);

    return (
        <Suspense
            fallback={
                <div className="p-6">
                    <div className="mx-auto max-w-lg">
                        <div className="space-y-4">
                            <div className="bg-muted h-6 w-48 animate-pulse rounded" />
                            <div className="bg-muted h-4 w-72 animate-pulse rounded" />
                            <div className="bg-muted h-10 w-full animate-pulse rounded" />
                        </div>
                    </div>
                </div>
            }
        >
            <EnrollStudentsContent classId={id} />
        </Suspense>
    );
}
