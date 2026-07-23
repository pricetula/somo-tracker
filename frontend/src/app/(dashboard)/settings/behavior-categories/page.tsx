/**
 * Behavior categories settings page — admin-only CRUD for behavior categories.
 */

import { getVerifiedRole } from "@/lib/auth-server";

export default async function BehaviorCategoriesPage() {
    const role = await getVerifiedRole();

    if (!role || (role !== "SCHOOL_ADMIN" && role !== "SYSTEM_ADMIN")) {
        return (
            <article>
                <p>You do not have access to this page.</p>
            </article>
        );
    }

    const { BehaviorCategoryManager } =
        await import("@/features/behavior/components/behavior-category-manager");

    return (
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-8 p-8">
            <div>
                <h1 className="font-semibold">Behavior Categories</h1>
                <div>
                    Manage the incident/behavior categories available in your school. Deactivating a
                    category preserves historical records.
                </div>
            </div>
            <BehaviorCategoryManager />

            {/* TODO: Add notifications placeholder */}
            {/*
             * TODO: In-app notifications for urgent behavior notes
             * Once a behavior note is approved with is_urgent = true, an in-app
             * notification should be generated for the parent. This will require:
             * - A notifications table
             * - A background job triggered on approval
             * - A bell icon component in the nav with unread badge
             */}
        </div>
    );
}
