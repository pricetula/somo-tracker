/**
 * Intercepted route for creating a new strand via the modal slot.
 *
 * Renders the CreateStrandDialog when navigating to /curriculum/new-strand
 * from within the curriculum detail page, preserving the background tree view.
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
                if (!open) router.back();
            }}
            learningAreaId={learningAreaId}
        />
    );
}
