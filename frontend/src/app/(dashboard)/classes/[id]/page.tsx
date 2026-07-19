/**
 * Class Detail Page — Full page render for /classes/:id.
 *
 * On hard refresh, this renders the class detail view directly.
 * When client-navigated from the classes table, it renders inside
 * the dashboard layout along with the modal slot.
 */

import { ClassDetailView } from "@/features/classes";

interface Props {
    params: Promise<{ id: string }>;
}

export default async function ClassDetailPage({ params }: Props) {
    const { id } = await params;
    return (
        <div className="p-6">
            <ClassDetailView classId={id} />
        </div>
    );
}
