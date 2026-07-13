/**
 * Intercepted route — Add stream rendered as a dialog overlay.
 *
 * Slides in as a centered modal when the user navigates to /streams/add
 * from within the app. On hard refresh the full page at /streams/add takes over.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { AddStreamForm } from "@/features/streams";

// ─── Props ────────────────────────────────────────────────────────────────

interface AddStreamModalProps {
    searchParams?: Promise<{ value?: string }>;
}

// ─── Modal ────────────────────────────────────────────────────────────────

export default function AddStreamModal(props: AddStreamModalProps) {
    const router = useRouter();
    const searchParams = props.searchParams ? use(props.searchParams) : {};
    const defaultValue = searchParams.value ?? "";

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Add Stream</DialogTitle>
                    <DialogDescription>
                        Add a new stream (section) for your school.
                    </DialogDescription>
                </DialogHeader>
                <AddStreamForm defaultValue={defaultValue} />
            </DialogContent>
        </Dialog>
    );
}
