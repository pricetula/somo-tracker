/**
 * Student report page — single student report (admin view).
 */

"use client";

import { useParams } from "next/navigation";

export default function StudentReportPage() {
    const params = useParams();
    const studentId = params.id as string;

    // TODO: Show compiled report for this student
    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Student Report</h1>
            <p className="text-muted-foreground">
                Report for student {studentId}. Select a term to view.
            </p>
        </div>
    );
}
