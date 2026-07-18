/**
 * Modal new indicator content.
 */

"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { CreateIndicatorDialog } from "@/features/curriculum";

export function ModalNewIndicatorContent() {
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
