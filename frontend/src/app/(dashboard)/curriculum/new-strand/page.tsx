/**
 * New Strand page.
 *
 * Full-page fallback when navigating directly to /curriculum/new-strand
 * or after a hard refresh (bypasses the intercepted modal route).
 */

"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { CreateStrandDialog } from "@/features/curriculum";

export default function NewStrandPage() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const learningAreaId = searchParams.get("learningAreaId") ?? "";

    return (
        <CreateStrandDialog
            open
            onOpenChange={(open: boolean) => {
                if (!open) router.push("/curriculum");
            }}
            learningAreaId={learningAreaId}
        />
    );
}
