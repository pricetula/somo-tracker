/**
 * Behavior note detail page — shows full note with review actions.
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function BehaviorNoteDetailPage() {
    const role = await getVerifiedRole();
    if (!role) return null;

    const { BehaviorNoteDetail } =
        await import("@/features/behavior/components/behavior-note-detail");
    return <BehaviorNoteDetail />;
}
