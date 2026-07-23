/**
 * Enrollment Page — Full page render for /classes/:id/enroll.
 *
 * Reads the `academictermid` query param.
 * - If present, passes it to EnrollStudentsPanel so the form uses that term.
 * - If absent, EnrollStudentsPanel shows academic year + term comboboxes.
 *
 * On hard refresh, this renders the enrollment panel directly.
 * When client-navigated from the class detail page, it renders as
 * a second-layer overlay (intercepted by @modal/(.)classes/[id]/enroll).
 */

"use client";

import { Suspense, use } from "react";
import { useSearchParams, useRouter } from "next/navigation";

import { EnrollStudentsPanel } from "@/features/classes";

interface Props {
    params: Promise<{ id: string }>;
}

function EnrollStudentsContent({ classId }: { classId: string }) {
    const router = useRouter();
    const searchParams = useSearchParams();
    const academicTermId = searchParams.get("academictermid") ?? undefined;

    return (
        <div className="p-6">
            <div className="mx-auto max-w-lg">
                {academicTermId ? (
                    <>
                        <h1 className="mb-1 text-lg font-semibold">Enroll Students</h1>
                        <p className="text-muted-foreground mb-6">
                            Search and select students to enroll in this class.
                        </p>
                    </>
                ) : (
                    <>
                        <h1 className="mb-1 text-lg font-semibold">Enroll Students</h1>
                        <p className="text-muted-foreground mb-6">
                            Select an academic year and term to enroll students into this class.
                        </p>
                    </>
                )}
                <EnrollStudentsPanel
                    classId={classId}
                    academicTermId={academicTermId}
                    onSuccess={() => router.back()}
                />
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
