/**
 * Route: /curriculum/[id]/strands/[strandId] — strand detail.
 */

import { StrandDetailPage } from "@/features/curriculum";

interface Props {
    params: Promise<{ id: string; strandId: string }>;
}

export default async function StrandDetailRoute({ params }: Props) {
    const { id, strandId } = await params;
    return <StrandDetailPage learningAreaId={id} strandId={strandId} />;
}
