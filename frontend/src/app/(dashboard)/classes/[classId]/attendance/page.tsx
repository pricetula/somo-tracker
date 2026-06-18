/**
 * Redirects /classes/[classId]/attendance → /classes/[classId]?tab=attendance
 */

import { redirect } from "next/navigation";

interface Props {
    params: Promise<{ classId: string }>;
}

export default async function AttendanceRedirect({ params }: Props) {
    const { classId } = await params;
    redirect(`/classes/${classId}?tab=attendance`);
}
