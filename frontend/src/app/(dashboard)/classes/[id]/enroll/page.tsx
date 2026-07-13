/**
 * Enrollment Page — Full page render for /classes/:id/enroll.
 *
 * On hard refresh, this renders the enrollment panel directly.
 * When client-navigated from the class detail page, it renders as
 * a second-layer overlay (intercepted by @modal/(.)classes/[id]/enroll).
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";

import { EnrollStudentsPanel } from "@/features/classes";
import { Button } from "@/components/ui/button";

interface Props {
    params: Promise<{ id: string }>;
}

export default function EnrollStudentsPage({ params }: Props) {
    const router = useRouter();
    const { id } = use(params);

    return (
        <div className="p-6">
            <Button variant="ghost" size="sm" onClick={() => router.back()} className="mb-4 -ml-2">
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back to Class
            </Button>
            <div className="mx-auto max-w-lg">
                <h1 className="mb-1 text-lg font-semibold">Enroll Students</h1>
                <p className="text-muted-foreground mb-6 text-sm">
                    Search and select students to enroll in this class.
                </p>
                <EnrollStudentsPanel classId={id} onSuccess={() => router.back()} />
            </div>
        </div>
    );
}
