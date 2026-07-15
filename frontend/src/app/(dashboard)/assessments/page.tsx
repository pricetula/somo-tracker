/**
 * Assessment Sessions listing page.
 *
 * Uses the shared DataTable component with status and type filters.
 * Click a session to view scores/grades and workflow actions.
 * The "Create" button navigates to /assessments/new (intercepted as a
 * dialog when navigated from this page).
 */

import { AssessmentSessionsList } from "@/features/assessments";

export default function SessionsPage() {
    return <AssessmentSessionsList />;
}
