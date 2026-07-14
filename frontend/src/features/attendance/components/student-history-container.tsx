/**
 * StudentHistoryContainer — reads student_id from params and renders history.
 */

"use client";

import { useParams } from "next/navigation";
import { StudentHistoryView } from "./student-history-view";

export function StudentHistoryContainer() {
    const params = useParams();
    const studentId = params.student_id as string;

    if (!studentId) {
        return (
            <div className="text-muted-foreground flex items-center justify-center py-16">
                <p>No student selected.</p>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Student Attendance History</h1>
            <StudentHistoryView studentId={studentId} />
        </div>
    );
}
