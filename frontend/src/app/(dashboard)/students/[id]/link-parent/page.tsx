/**
 * Full-page fallback for linking a parent to a student.
 *
 * Renders when the user navigates directly to /students/:id/link-parent
 * (e.g. on hard refresh). The intercepted modal at
 * @modal/(.)students/[id]/link-parent takes precedence in the parallel slot.
 */

"use client";

import { use, useCallback } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { LinkParentForm } from "@/features/parents";

interface Props {
    params: Promise<{ id: string }>;
}

export default function LinkParentPage({ params }: Props) {
    const router = useRouter();
    const { id } = use(params);

    const handleSuccess = useCallback(() => {
        router.push(`/students/${id}`);
    }, [router, id]);

    const handleCancel = useCallback(() => {
        router.push(`/students/${id}`);
    }, [router, id]);

    return (
        <div className="mx-auto max-w-lg py-8">
            <Button variant="ghost" size="sm" onClick={handleCancel} className="mb-4">
                <ArrowLeft className="mr-1.5 size-4" />
                Students
            </Button>

            <Card>
                <CardHeader>
                    <CardTitle>Link Parent</CardTitle>
                    <CardDescription>
                        Search and select a parent to link to this student.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <LinkParentForm
                        studentId={id}
                        onSuccess={handleSuccess}
                        onCancel={handleCancel}
                    />
                </CardContent>
            </Card>
        </div>
    );
}
