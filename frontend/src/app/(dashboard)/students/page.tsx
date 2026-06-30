/**
 * Students listing page — active enrolled students.
 *
 * Maps to GET /api/v1/students/list.
 */

"use client";

import * as React from "react";

import { StudentsTable, useStudents } from "@/features/students";

export default function StudentsPage() {
    const {
        data: studentsData,
        isLoading: studentsLoading,
        isError: studentsError,
    } = useStudents();

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Students</h1>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-col">
                    {studentsError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load students. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <StudentsTable
                                students={studentsData?.students ?? []}
                                total={studentsData?.total ?? 0}
                                isLoading={studentsLoading}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
