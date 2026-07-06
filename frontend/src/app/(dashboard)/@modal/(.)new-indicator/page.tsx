/**
 * Intercepted route for creating a new performance indicator via the modal slot.
 *
 * Renders the CreateIndicatorDialog when navigating to /curriculum/new-indicator
 * from within the curriculum detail page, preserving the background tree view.
 */

"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { CreateIndicatorDialog } from "@/features/curriculum";

export default function NewIndicatorPage() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const subStrandId = searchParams.get("subStrandId") ?? "";

    return (
        <CreateIndicatorDialog
            open
            onOpenChange={(open: boolean) => {
                if (!open) router.back();
            }}
            subStrandId={subStrandId}
        />
    );
}
