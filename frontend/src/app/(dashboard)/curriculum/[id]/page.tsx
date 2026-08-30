/**
 * Route: /curriculum/[id] — learning area detail.
 */

import { CurriculumDetailPage } from "@/features/curriculum";

interface Props {
    params: Promise<{ id: string }>;
}

export default async function CurriculumDetailRoute({ params }: Props) {
    const { id } = await params;
    return <CurriculumDetailPage learningAreaId={id} />;
}
