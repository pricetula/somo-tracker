/**
 * Grading Profile Detail page — Full page render for /grading/:id.
 *
 * On hard refresh, renders the profile detail directly with range management.
 * When client-navigated from the grading listing, it is intercepted
 * by @modal/(.)grading/[id] and rendered as a side sheet.
 */

import { ScaleProfileDetailView } from "@/features/assessments";

interface Props {
    params: Promise<{ id: string }>;
}

export default async function ScaleProfileDetailPage({ params }: Props) {
    const { id } = await params;
    return (
        <div className="p-6">
            <ScaleProfileDetailView profileId={id} />
        </div>
    );
}
