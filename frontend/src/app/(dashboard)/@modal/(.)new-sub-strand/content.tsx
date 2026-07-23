/**
 * Modal new sub-strand content.
 */

"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { CreateSubStrandDialog } from "@/features/curriculum";

export function ModalNewSubStrandContent() {
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
