/**
 * New Sub-Strand page.
 *
 * Full-page fallback when navigating directly to /curriculum/new-sub-strand
 * or after a hard refresh (bypasses the intercepted modal route).
 */

"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { CreateSubStrandDialog } from "@/features/curriculum";

export default function NewSubStrandPage() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const strandId = searchParams.get("strandId") ?? "";

    return (
        <CreateSubStrandDialog
            open
            onOpenChange={(open: boolean) => {
                if (!open) router.push("/curriculum");
            }}
            strandId={strandId}
        />
    );
}
