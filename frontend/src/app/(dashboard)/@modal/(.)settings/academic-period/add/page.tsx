"use client";

import { useRouter } from "next/navigation";
import { AcademicPeriodAddForm } from "@/features/academic-period";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function AcademicPeriodAddModalPage() {
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Create Academic Period</DialogTitle>
                </DialogHeader>
                <AcademicPeriodAddForm />
            </DialogContent>
        </Dialog>
    );
}
