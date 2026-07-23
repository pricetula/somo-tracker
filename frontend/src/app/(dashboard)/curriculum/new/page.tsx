/**
 * New Learning Area page.
 *
 * Full-page fallback when navigating directly to /curriculum/new
 * or after a hard refresh (bypasses the intercepted modal route).
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
                if (!open) router.push("/curriculum");
            }}
        />
    );
}
