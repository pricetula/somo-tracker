/**
 * Timetable page.
 *
 * Displays the weekly timetable grid with period structure and lesson allocations.
 * Requires SCHOOL_ADMIN, TEACHER, NURSE, or FINANCE role.
 * Read-only for TEACHER, NURSE, FINANCE. Editable for SCHOOL_ADMIN.
 *
 * Maps to:
 *   GET /api/v1/timetable/structure?academic_year_id=
 *   GET /api/v1/timetable/slots/enriched?academic_year_id=
 */

"use client";

import { useEffect, useState } from "react";
import { getCurrentYearAndTerm } from "@/lib/api/academic-terms";
import { useMe } from "@/hooks/use-auth";
import { TimetableView } from "@/features/timetable";

export default function TimetablePage() {
    const { data: me, isLoading: meLoading } = useMe();
    const [academicYearId, setAcademicYearId] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    // Determine read-only access: TEACHER, NURSE, FINANCE are read-only
    // SCHOOL_ADMIN and SYSTEM_ADMIN can edit
    const isReadOnly = me?.role && ["TEACHER", "NURSE", "FINANCE"].includes(me.role);

    useEffect(() => {
        async function loadCurrentYear() {
            try {
                const data = await getCurrentYearAndTerm();
                if (data.academic_year_id) {
                    setAcademicYearId(data.academic_year_id);
                } else {
                    setError("No active academic year found. Please contact an administrator.");
                }
            } catch (err) {
                console.error("Failed to load current academic year:", err);
                setError("Failed to load timetable. Please try again.");
            } finally {
                setIsLoading(false);
            }
        }

        loadCurrentYear();
    }, []);

    if (meLoading || isLoading) {
        return (
            <div className="flex h-[60vh] items-center justify-center">
                <div className="animate-pulse space-y-4">
                    <div className="bg-muted h-8 w-1/4 rounded" />
                    <div className="bg-muted h-64 rounded" />
                </div>
            </div>
        );
    }

    if (error || !academicYearId) {
        return (
            <div className="flex h-[60vh] items-center justify-center">
                <div className="text-center">
                    <p className="text-destructive">Unable to load timetable</p>
                    <p className="text-muted-foreground mt-2 text-sm">
                        {error ?? "No active academic year"}
                    </p>
                </div>
            </div>
        );
    }

    return <TimetableView academicYearId={academicYearId} isReadOnly={isReadOnly} />;
}
