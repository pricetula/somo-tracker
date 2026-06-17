"use client";

import { useRouter } from "next/navigation";
import { Dialog, DialogContent, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { ClassStreamGenerator } from "@/features/classes";

export default function ClassesGenerateModal() {
    const router = useRouter();

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl" showCloseButton>
                <DialogTitle className="sr-only">Establish Classes &amp; Streams</DialogTitle>
                <DialogDescription className="sr-only">
                    Define your streams and generate classrooms
                </DialogDescription>
                <ClassStreamGenerator onSuccess={() => router.back()} />
            </DialogContent>
        </Dialog>
    );
}
