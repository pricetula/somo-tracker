/**
 * Assessment root page — redirects to blueprints list.
 */

import { redirect } from "next/navigation";

export default function AssessmentPage() {
    redirect("/assessment/blueprints");
}
