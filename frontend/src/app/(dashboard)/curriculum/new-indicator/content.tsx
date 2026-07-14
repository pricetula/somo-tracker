/**
 * New Indicator page content — client component with useSearchParams.
 */

"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { CreateIndicatorDialog } from "@/features/curriculum";

export function NewIndicatorPageContent() {
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
