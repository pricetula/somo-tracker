/**
 * Intercepted route for creating a new learning area via the modal slot.
 *
 * This renders the CreateLearningAreaDialog when navigating to /curriculum/new
 * from within the dashboard, preserving the background page.
 */

"use client";

import { useRouter } from "next/navigation";
import { CreateLearningAreaDialog } from "@/features/curriculum";

export default function CurriculumNewPage() {
    const router = useRouter();

    return (
        <CreateLearningAreaDialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        />
    );
}
