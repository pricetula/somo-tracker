/**
 * Nurse Detail Page — Full page render for /nurses/:id.
 *
 * On hard refresh, this renders the nurse detail view directly.
 * When client-navigated from the nurses table, it renders inside
 * the dashboard layout along with the modal slot.
 */

import { NurseDetail } from "@/features/nurses";

interface Props {
    params: Promise<{ id: string }>;
}

export default async function NurseDetailPage({ params }: Props) {
    const { id } = await params;
    return (
        <div className="p-6">
            <NurseDetail id={id} />
        </div>
    );
}
