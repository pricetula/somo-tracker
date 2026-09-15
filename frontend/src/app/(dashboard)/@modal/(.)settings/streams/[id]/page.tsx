"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { StreamForm } from "@/features/streams";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function StreamDetailModal({ params }: { params: Promise<{ id: string }> }) {
    const { id } = use(params);
    const router = useRouter();

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="max-h-[85vh] max-w-xl overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Edit Stream</DialogTitle>
                </DialogHeader>
                <StreamForm streamId={id} onSuccess={() => router.back()} />
            </DialogContent>
        </Dialog>
    );
}
