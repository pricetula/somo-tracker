"use client";

import { useRouter } from "next/navigation";
import { CreateUsingStreamsForm } from "@/features/classes/components/create-using-streams-form";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function CreateUsingStreamsModalPage() {
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-3xl md:max-w-5xl">
                <DialogHeader>
                    <DialogTitle>Auto populate Classrooms</DialogTitle>
                </DialogHeader>
                <CreateUsingStreamsForm />
            </DialogContent>
        </Dialog>
    );
}
