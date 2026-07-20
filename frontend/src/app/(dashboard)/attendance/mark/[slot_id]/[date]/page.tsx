/**
 * Full-page route — Mark attendance for a specific timetable slot on a date.
 *
 * This page renders standalone on hard refresh / direct navigation.
 * When navigated from within the dashboard, the @modal parallel route
 * intercepts it and renders as a sliding sheet instead.
 */

import { notFound } from "next/navigation";
import { getVerifiedRole } from "@/lib/auth-server";
import { AttendanceMarkPage } from "@/features/attendance/components/attendance-mark-page";

interface Props {
    params: Promise<{ slot_id: string; date: string }>;
}

export default async function Page({ params }: Props) {
    const { slot_id, date } = await params;
    const role = await getVerifiedRole();

    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    const allowedRoles = ["TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"];
    if (!allowedRoles.includes(role)) {
        return (
            <article>
                <p>You do not have access to this page.</p>
            </article>
        );
    }

    if (!slot_id || !date) {
        notFound();
    }

    return (
        <div className="mx-auto max-w-2xl space-y-6">
            <AttendanceMarkPage slotId={slot_id} date={date} />
        </div>
    );
}
