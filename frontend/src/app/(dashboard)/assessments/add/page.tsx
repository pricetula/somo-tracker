/**
 * Create Assessment Session page — Full page render for /assessments/add.
 *
 * On hard refresh, renders the create session form as a standalone page.
 * When client-navigated from the sessions listing, it is intercepted
 * by @modal/(.)assessments/add and rendered as a dialog overlay.
 */

import { CreateAssessmentSessionForm } from "@/features/assessments";

export default function AddAssessmentPage() {
    return (
        <div className="mx-auto max-w-lg p-6">
            <h1 className="mb-1 text-lg font-semibold">Create Assessment Session</h1>
            <p className="text-muted-foreground mb-6 text-sm">
                Create a new assessment session. Choose between marks-based (quantitative) grading
                or rubric (indicator-level) grading.
            </p>
            <CreateAssessmentSessionForm />
        </div>
    );
}
