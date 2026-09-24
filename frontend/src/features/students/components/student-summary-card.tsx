"use client";

import { useStudentSummary } from "../hooks/use-student-summary";

export function StudentSummaryCard() {
    const { data: summary, isLoading, isError } = useStudentSummary();

    if (isLoading) return <div>Loading...</div>;

    if (isError) return <div>Error loading summary.</div>;

    if (!summary) return null;

    const malePercentage = (summary.male_count / summary.total_students) * 100;
    const femalePercentage = (summary.female_count / summary.total_students) * 100;

    return (
        <article>
            <header className="mb-4 space-y-2">
                <h2 className="text-xl font-bold">{summary.total_students} Students</h2>
                <div className="flex w-40 items-center">
                    <div className="mr-2">Male</div>
                    <div
                        className={`h-1 rounded-l-2xl bg-blue-400`}
                        style={{ width: malePercentage }}
                    ></div>
                    <div
                        className={`h-1 rounded-r-2xl bg-blue-600`}
                        style={{ width: femalePercentage }}
                    ></div>
                    <div className="ml-2">Female</div>
                </div>
            </header>
            <p>{summary.unassigned_count} Unassigned</p>
            <p>{summary.unlinked_guardians_count} Unlinked Guardians</p>
        </article>
    );
}
