/**
 * Student Detail page.
 *
 * Shows profile card + enrollment timeline with actions.
 * Maps to GET /api/v1/students/:id.
 */

"use client";

import * as React from "react";
import { useParams, useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { ArrowLeft, Pencil } from "lucide-react";

import {
    StudentProfileCard,
    EnrollmentTimeline,
    EnrollDialog,
    useStudentDetail,
    useUpdateStudent,
} from "@/features/students";

export default function StudentDetailPage() {
    const params = useParams();
    const router = useRouter();
    const id = params.id as string;

    const { data: detailData, isLoading, isError } = useStudentDetail(id);

    const updateStudent = useUpdateStudent();
    const [enrollDialogOpen, setEnrollDialogOpen] = React.useState(false);

    const detail = detailData?.data;

    const handleToggleActive = async () => {
        if (!detail) return;
        await updateStudent.mutateAsync({
            id,
            data: { is_active: !detail.is_active },
        });
    };

    if (isError || (!isLoading && !detail)) {
        return (
            <div className="flex flex-col items-center justify-center py-16">
                <p className="text-destructive text-sm font-medium">Student not found</p>
                <Button
                    variant="outline"
                    size="sm"
                    className="mt-4"
                    onClick={() => router.push("/students")}
                >
                    Back to Students
                </Button>
            </div>
        );
    }

    return (
        <div className="flex flex-1 flex-col px-6 pt-6 pb-8">
            {/* Back link */}
            <Button
                variant="ghost"
                size="sm"
                className="mb-4 w-fit"
                onClick={() => router.push("/students")}
            >
                <ArrowLeft className="mr-1.5 size-4" />
                Back to Students
            </Button>

            {/* Page header + actions */}
            <div className="mb-6 flex items-start justify-between">
                <div>
                    <h1 className="text-2xl font-semibold tracking-tight">
                        {detail?.full_name ?? "Student Detail"}
                    </h1>
                    {detail?.class_name && (
                        <p className="text-muted-foreground mt-0.5 text-sm">{detail.class_name}</p>
                    )}
                </div>

                <div className="flex items-center gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => router.push(`/students/${id}/edit`)}
                    >
                        <Pencil className="mr-1.5 size-3.5" />
                        Edit
                    </Button>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={handleToggleActive}
                        disabled={updateStudent.isPending}
                    >
                        {detail?.is_active ? "Deactivate" : "Reactivate"}
                    </Button>
                </div>
            </div>

            {/* Two-column layout */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                {/* Profile Card */}
                <StudentProfileCard detail={detail} isLoading={isLoading} />

                {/* Enrollment Timeline */}
                <EnrollmentTimeline
                    enrollments={detail?.enrollments ?? []}
                    isLoading={isLoading}
                    onEnrollClick={() => setEnrollDialogOpen(true)}
                />
            </div>

            {/* Enroll Dialog */}
            <EnrollDialog
                open={enrollDialogOpen}
                onOpenChange={setEnrollDialogOpen}
                studentId={id}
            />
        </div>
    );
}
