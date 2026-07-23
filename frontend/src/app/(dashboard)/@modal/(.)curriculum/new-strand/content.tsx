/**
 * Modal new strand content.
 */

"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { CreateStrandDialog } from "@/features/curriculum";

export function ModalNewStrandContent() {
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
