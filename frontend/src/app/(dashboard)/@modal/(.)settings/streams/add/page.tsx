"use client";

import { useRouter } from "next/navigation";
import { StreamForm } from "@/features/streams";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function AddStreamModal() {
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
                    <DialogTitle>Add Stream</DialogTitle>
                </DialogHeader>
                <StreamForm onSuccess={() => router.back()} />
            </DialogContent>
        </Dialog>
    );
}
