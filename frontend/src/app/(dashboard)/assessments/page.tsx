/**
 * Assessments page — role-agnostic route.
 *
 * TEACHER / SCHOOL_ADMIN: Assessment sessions listing with DataTable
 * PARENT: view published results and term report cards for children
 */

import { getVerifiedRole } from "@/lib/auth-server";
import { AssessmentSessionsList, ParentAssessmentsView } from "@/features/assessments";

export default async function AssessmentsPage() {
    const role = await getVerifiedRole();

    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    if (role === "PARENT") {
        return <ParentAssessmentsView />;
    }

    return <AssessmentSessionsList />;
}
