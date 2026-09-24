"use client";

import { useRouter } from "next/navigation";
import { StudentsImportOrchestrator } from "@/features/students";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function StudentAddModalPage() {
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
                    <DialogTitle>Add Student</DialogTitle>
                </DialogHeader>
                <StudentsImportOrchestrator />
            </DialogContent>
        </Dialog>
    );
}
