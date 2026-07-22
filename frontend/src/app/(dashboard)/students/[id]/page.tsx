/**
 * Student Detail Page — Full page render for /students/:id.
 *
 * Shows student profile, enrollment history, behavior notes,
 * health information, and report links.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";

import { StudentDetailContent } from "@/features/students";

interface Props {
    params: Promise<{ id: string }>;
}

export default function StudentDetailPage({ params }: Props) {
    const router = useRouter();
    const { id } = use(params);

    return (
        <StudentDetailContent
            studentId={id}
            variant="page"
            onDeleteSuccess={() => router.push("/students")}
        />
    );
}
