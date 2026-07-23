/**
 * Intercepted route — Create assessment session rendered as a dialog overlay.
 *
 * Slides in as a centered modal when the teacher clicks "Create"
 * from the assessment sessions listing page.
 * On hard refresh the full page at /assessments/add takes over.
 */

"use client";

import { useRouter } from "next/navigation";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { CreateAssessmentSessionForm } from "@/features/assessments";

export default function AddAssessmentModal() {
    const router = useRouter();

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="sm:max-w-lg">
                <DialogHeader>
                    <DialogTitle>Create Assessment Session</DialogTitle>
                    <DialogDescription>
                        Choose marks-based (quantitative) or rubric (indicator-level) grading.
                    </DialogDescription>
                </DialogHeader>
                <CreateAssessmentSessionForm />
            </DialogContent>
        </Dialog>
    );
}
