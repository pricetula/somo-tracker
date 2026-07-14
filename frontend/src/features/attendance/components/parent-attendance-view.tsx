/**
 * ParentAttendanceView — shows attendance for the parent's linked children.
 * If multiple children, shows a selector to switch between them.
 */

"use client";

import { useState } from "react";
import { useMe } from "@/hooks/use-auth";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { ParentAttendanceSummary } from "./parent-attendance-summary";

// TODO: Replace with actual API call to fetch parent's linked children.
// The real implementation should call GET /api/v1/parents/children
// to get the list of students linked to this parent account.

interface ChildInfo {
    id: string;
    full_name: string;
}

const MOCK_CHILDREN: ChildInfo[] = [];
const MOCK_TERM_ID = "00000000-0000-0000-0000-000000000000";

export function ParentAttendanceView() {
    const { data: me } = useMe();
    const [selectedStudentId, setSelectedStudentId] = useState<string>("");

    // TODO: fetch actual children for this parent
    // const { data: children } = useParentChildren();
    const children = MOCK_CHILDREN;

    if (!me) return null;

    if (children.length === 0) {
        return (
            <div className="text-muted-foreground flex items-center justify-center py-16">
                <p>No children linked to your account.</p>
            </div>
        );
    }

    const effectiveStudentId = selectedStudentId || children[0]?.id;

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
                                <SelectItem key={child.id} value={child.id}>
                                    {child.full_name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
            )}

            {effectiveStudentId && (
                <ParentAttendanceSummary studentId={effectiveStudentId} termId={MOCK_TERM_ID} />
            )}
        </div>
    );
}
