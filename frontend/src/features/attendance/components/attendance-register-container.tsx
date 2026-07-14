/**
 * AttendanceRegisterContainer — reads query params and renders the roster
 * for a specific slot/date. Used as drill-down from admin dashboard.
 */

"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ClipboardList } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TeacherAttendanceRoster } from "./teacher-attendance-roster";
import { AttendanceEmptyState } from "./attendance-empty-state";

interface AttendanceRegisterContainerProps {
    role: string;
}

export function AttendanceRegisterContainer({ role }: AttendanceRegisterContainerProps) {
    const searchParams = useSearchParams();
    const slotId = searchParams.get("slot_id");
    const date = searchParams.get("date") ?? undefined;

    if (!slotId) {
        return (
            <AttendanceEmptyState
                icon={ClipboardList}
                title="No slot selected"
                description="Select a class period from the attendance dashboard to view or mark attendance."
            >
                <Button variant="outline" size="sm" asChild>
                    <Link href="/attendance">Go to dashboard</Link>
                </Button>
            </AttendanceEmptyState>
        );
    }

    // Admins have elevated scope (no same-day restriction)
    const isAdmin = role === "SCHOOL_ADMIN" || role === "SYSTEM_ADMIN";

    return (
        <div className="space-y-6">
            <TeacherAttendanceRoster timetableSlotId={slotId} date={date} isLocked={!isAdmin} />
        </div>
    );
}
