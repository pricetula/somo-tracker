/**
 * StudentHistoryContainer — reads student_id from params and renders history.
 */

"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { UserX } from "lucide-react";
import { Button } from "@/components/ui/button";
import { StudentHistoryView } from "./student-history-view";
import { AttendanceEmptyState } from "./attendance-empty-state";

export function StudentHistoryContainer() {
    const params = useParams();
    const studentId = params.student_id as string;

    if (!studentId) {
        return (
            <AttendanceEmptyState
                icon={UserX}
                title="No student selected"
                description="Select a student from the student list to view their attendance history."
            >
                <Button variant="outline" size="sm" asChild>
                    <Link href="/students">Browse students</Link>
                </Button>
            </AttendanceEmptyState>
        );
    }

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Student Attendance History</h1>
            <StudentHistoryView studentId={studentId} />
        </div>
    );
}
