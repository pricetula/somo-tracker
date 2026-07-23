/**
 * Intercepted route — Link Parent overlay as a dialog.
 *
 * Renders on top of the student detail page when the teacher clicks
 * "Link Parent". On hard refresh, the full page at /students/:id/link-parent
 * takes over.
 */

"use client";

import { use, useCallback } from "react";
import { useRouter } from "next/navigation";

import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { LinkParentForm } from "@/features/parents";

interface Props {
    params: Promise<{ id: string }>;
}

export default function LinkParentModal({ params }: Props) {
    const router = useRouter();
    const { id } = use(params);

    const handleClose = useCallback(() => {
        router.back();
    }, [router]);

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="max-w-md">
                <DialogHeader>
                    <DialogTitle>Link Parent</DialogTitle>
                    <DialogDescription>
                        Search and select a parent to link to this student.
                    </DialogDescription>
                </DialogHeader>
                <LinkParentForm studentId={id} onSuccess={handleClose} onCancel={handleClose} />
            </DialogContent>
        </Dialog>
    );
}
