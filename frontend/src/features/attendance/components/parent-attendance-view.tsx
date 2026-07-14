/**
 * ParentAttendanceView — shows attendance for the parent's linked children.
 * If multiple children, shows a selector to switch between them.
 *
 * Fetches children from GET /api/v1/parents/me and the current term from
 * GET /api/v1/academic-terms with is_current filter.
 */

"use client";

import { useState, useMemo } from "react";
import { UserX, AlertCircle } from "lucide-react";
import { useMe } from "@/hooks/use-auth";
import { getMyParentProfile } from "@/lib/api/parents";
import { useAcademicTerms } from "@/features/academic-terms/hooks/use-academic-terms";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { ParentAttendanceSummary } from "./parent-attendance-summary";
import { AttendanceEmptyState } from "./attendance-empty-state";

export function ParentAttendanceView() {
    const { data: me } = useMe();

    // Fetch parent's own profile with linked children
    const { data: parentProfile, isLoading: profileLoading } = useQuery({
        queryKey: ["parent", "me"],
        queryFn: () => getMyParentProfile(),
        enabled: !!me,
    });

    // Fetch all terms to find the current one
    const { data: termsData, isLoading: termsLoading } = useAcademicTerms();

    const [selectedStudentId, setSelectedStudentId] = useState<string>("");

    const children = parentProfile?.data?.linked_students ?? [];

    // Find the current term — the one with is_current = true
    const currentTerm = useMemo(() => {
        return termsData?.items?.find((t) => t.is_current) ?? null;
    }, [termsData]);

    // Auto-select first child when data loads
    const effectiveStudentId = selectedStudentId || children[0]?.student_id || "";

    if (!me) return null;

    if (profileLoading || termsLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-24 w-full rounded-lg" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
            </div>
        );
    }

    if (children.length === 0) {
        return (
            <AttendanceEmptyState
                icon={UserX}
                title="No children linked to your account"
                description="To view attendance, your school admin needs to link your child's record to your parent account."
            >
                <Button variant="outline" size="sm">
                    Contact school admin
                </Button>
            </AttendanceEmptyState>
        );
    }

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Attendance</h1>

            {children.length > 1 && (
                <div className="w-64">
                    <Select value={effectiveStudentId} onValueChange={setSelectedStudentId}>
                        <SelectTrigger>
                            <SelectValue placeholder="Select child..." />
                        </SelectTrigger>
                        <SelectContent>
                            {children.map((child) => (
                                <SelectItem key={child.student_id} value={child.student_id}>
                                    {child.full_name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
            )}

            {effectiveStudentId && currentTerm ? (
                <ParentAttendanceSummary studentId={effectiveStudentId} termId={currentTerm.id} />
            ) : effectiveStudentId ? (
                <AttendanceEmptyState
                    icon={AlertCircle}
                    title="No active term"
                    description="The current academic term has not been set. Attendance data will appear once a term is activated."
                >
                    <Button variant="outline" size="sm">
                        Contact school admin
                    </Button>
                </AttendanceEmptyState>
            ) : null}
        </div>
    );
}
