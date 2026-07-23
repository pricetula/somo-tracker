/**
 * Edit Student page.
 *
 * Pre-populated form with existing demographics.
 * Maps to PUT /api/v1/students/:id.
 */

"use client";

import { useParams, useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

import { StudentForm, useStudentDetail } from "@/features/students";

export default function EditStudentPage() {
    const params = useParams();
    const router = useRouter();
    const id = params.id as string;

    const { data: detailData, isLoading, isError } = useStudentDetail(id);

    if (isLoading) {
        return (
            <div className="mx-auto flex max-w-xl flex-col px-6 pt-6 pb-8">
                <Skeleton className="mb-4 h-8 w-32" />
                <Skeleton className="mb-6 h-6 w-48" />
                <Skeleton className="h-96 w-full" />
            </div>
        );
    }

    if (isError || !detailData?.data) {
        return (
            <div className="flex flex-col items-center justify-center py-16">
                <p className="text-destructive font-medium">Student not found</p>
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

    const handleSuccess = () => {
        router.push(`/students/${id}`);
    };

    return <StudentForm mode="edit" initialData={detailData.data} onSuccess={handleSuccess} />;
}
