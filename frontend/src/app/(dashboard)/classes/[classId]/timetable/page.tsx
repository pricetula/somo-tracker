/**
 * Redirects /classes/[classId]/timetable → /classes/[classId]?tab=timetable
 */

import { redirect } from "next/navigation";

interface Props {
    params: Promise<{ classId: string }>;
}

export default async function TimetableRedirect({ params }: Props) {
    const { classId } = await params;
    redirect(`/classes/${classId}?tab=timetable`);
}
