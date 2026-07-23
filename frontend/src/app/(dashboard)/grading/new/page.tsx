/**
 * Create Scale Profile page — Full page render for /grading/new.
 *
 * On hard refresh, renders the create profile form as a standalone page.
 * When client-navigated from the grading listing, it is intercepted
 * by @modal/(.)grading/new and rendered as a dialog overlay.
 */

import { CreateScaleProfileForm } from "@/features/assessments";

export default function CreateScaleProfilePage() {
    return (
        <div className="mx-auto max-w-lg p-6">
            <h1 className="mb-1 text-lg font-semibold">Create Scale Profile</h1>
            <p className="text-muted-foreground mb-6">
                Define a new set of percentage-to-CBC-level conversion rules. After creation, you
                will set up the percentage ranges.
            </p>
            <CreateScaleProfileForm />
        </div>
    );
}
