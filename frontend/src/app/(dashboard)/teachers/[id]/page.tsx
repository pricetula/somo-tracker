/**
 * Teacher Detail Page — Full page render for /teachers/:id.
 *
 * On hard refresh, this renders the teacher detail view directly.
 * When client-navigated from the teachers table, it renders inside
 * the dashboard layout along with the modal slot.
 */

import { TeacherDetail } from "@/features/teachers";

interface Props {
    params: Promise<{ id: string }>;
}

export default async function TeacherDetailPage({ params }: Props) {
    const { id } = await params;
    return (
        <div className="p-6">
            <TeacherDetail id={id} />
        </div>
    );
}
