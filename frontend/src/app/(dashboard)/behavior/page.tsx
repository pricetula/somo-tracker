/**
 * Behavior page — role-agnostic route.
 *
 * SCHOOL_ADMIN: shows the review queue
 * TEACHER: shows their own submitted notes
 * PARENT: shows approved notes for their children
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function BehaviorPage() {
    const role = await getVerifiedRole();

    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    switch (role) {
        case "SCHOOL_ADMIN":
        case "SYSTEM_ADMIN": {
            const { BehaviorReviewQueue } =
                await import("@/features/behavior/components/behavior-review-queue");
            return (
                <div className="space-y-6">
                    <h1 className="text-2xl font-bold">Behavior</h1>
                    <BehaviorReviewQueue />
                </div>
            );
        }
        case "TEACHER": {
            const { TeacherBehaviorView } =
                await import("@/features/behavior/components/teacher-behavior-view");
            return (
                <div className="space-y-6">
                    <h1 className="text-2xl font-bold">My Behavior Notes</h1>
                    <TeacherBehaviorView />
                </div>
            );
        }
        case "PARENT": {
            // TODO: Show approved behavior notes for linked children
            return (
                <div className="space-y-6">
                    <h1 className="text-2xl font-bold">Behavior</h1>
                    <p className="text-muted-foreground">
                        Behavior notes are available in your child&apos;s term report.
                    </p>
                </div>
            );
        }
        default:
            return (
                <article>
                    <p>You do not have access to this page.</p>
                </article>
            );
    }
}
