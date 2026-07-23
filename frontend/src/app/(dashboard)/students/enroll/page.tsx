/**
 * Full-page fallback for batch enrollment.
 *
 * Renders when the user navigates directly to /students/enroll
 * (e.g. on hard refresh or JS disabled). The intercepted modal
 * at @modal/(.)students/enroll takes precedence in the parallel slot.
 */

"use client";

import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { BatchEnrollForm } from "@/features/students";

export default function StudentsEnrollPage() {
    const router = useRouter();

    return (
        <div className="mx-auto max-w-lg py-8">
            <Button
                variant="ghost"
                size="sm"
                onClick={() => router.push("/students")}
                className="mb-4"
            >
                <ArrowLeft className="mr-1.5 size-4" />
                Back to Students
            </Button>

            <Card>
                <CardHeader>
                    <CardTitle>Enroll Students</CardTitle>
                    <CardDescription>
                        Select a class to enroll the selected students in the current academic term.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <BatchEnrollForm
                        onSuccess={() => router.push("/students")}
                        onCancel={() => router.push("/students")}
                    />
                </CardContent>
            </Card>
        </div>
    );
}
