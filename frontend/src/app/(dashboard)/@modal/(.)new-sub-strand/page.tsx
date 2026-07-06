/**
 * Intercepted route for creating a new sub-strand via the modal slot.
 *
 * Renders the CreateSubStrandDialog when navigating to /curriculum/new-sub-strand
 * from within the curriculum detail page, preserving the background tree view.
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
                if (!open) router.back();
            }}
            strandId={strandId}
        />
    );
}
