/**
 * Grading Scale Profiles listing page — Admin only.
 *
 * Uses the shared DataTable component for listing. Click a profile to
 * view/edit its percentage ranges. The "Add Profile" button navigates
 * to /grading/new (intercepted as a dialog when navigated from this page).
 */

import { GradingScaleProfilesList } from "@/features/assessments";

export default function GradingProfilesPage() {
    return <GradingScaleProfilesList />;
}
