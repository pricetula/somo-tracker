/**
 * New Sub-Strand page content — client component with useSearchParams.
 */

"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { CreateSubStrandDialog } from "@/features/curriculum";

export function NewSubStrandPageContent() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const strandId = searchParams.get("strandId") ?? "";

    return (
        <CreateSubStrandDialog
            open
            onOpenChange={(open: boolean) => {
                if (!open) router.back();
            }}
            strandId={strandId}
        />
    );
}
