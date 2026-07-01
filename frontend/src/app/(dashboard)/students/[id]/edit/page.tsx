/**
 * Edit Student page.
 *
 * Pre-populated form with existing demographics.
 * Maps to PUT /api/v1/students/:id.
 */

"use client";

import * as React from "react";
import { useParams, useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft } from "lucide-react";

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

    const handleSuccess = () => {
        router.push(`/students/${id}`);
    };

    return (
        <div className="mx-auto flex max-w-xl flex-col px-6 pt-6 pb-8">
            <Button
                variant="ghost"
                size="sm"
                className="mb-4 w-fit"
                onClick={() => router.push(`/students/${id}`)}
            >
                <ArrowLeft className="mr-1.5 size-4" />
                Back to Profile
            </Button>

            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">Edit Student</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Update {detailData.data.full_name}&apos;s demographic information.
                </p>
            </div>

            <StudentForm mode="edit" initialData={detailData.data} onSuccess={handleSuccess} />
        </div>
    );
}
