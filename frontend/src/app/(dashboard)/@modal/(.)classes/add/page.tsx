"use client";

import { useRouter } from "next/navigation";
import { ClassAddForm } from "@/features/classes";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function ClassAddModalPage() {
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] overflow-y-auto md:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Add Class</DialogTitle>
                </DialogHeader>
                <ClassAddForm />
            </DialogContent>
        </Dialog>
    );
}
