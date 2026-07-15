/**
 * Assessment Session Detail page — Full page render for /assessments/:id.
 *
 * Shows the session info, scores/grades, and workflow actions.
 * In DRAFT, the teacher can submit for approval.
 * In PENDING_APPROVAL, the admin can approve or reject.
 * PUBLISHED sessions are read-only.
 */

import { AssessmentSessionDetailView } from "@/features/assessments";

interface Props {
    params: Promise<{ id: string }>;
}

export default async function AssessmentDetailPage({ params }: Props) {
    const { id } = await params;
    return (
        <div className="p-6">
            <AssessmentSessionDetailView sessionId={id} />
        </div>
    );
}
