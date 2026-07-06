/**
 * New Performance Indicator page.
 *
 * Full-page fallback when navigating directly to /curriculum/new-indicator
 * or after a hard refresh (bypasses the intercepted modal route).
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
                if (!open) router.push("/curriculum");
            }}
            subStrandId={subStrandId}
        />
    );
}
