"use client";

import { useRouter, useParams } from "next/navigation";
import { ClassDetail } from "@/features/classes";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function ClassDetailModalPage() {
    const router = useRouter();
    const params = useParams();
    const id = params.id as string;

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] max-w-2xl overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Class Details</DialogTitle>
                </DialogHeader>
                <ClassDetail id={id} />
            </DialogContent>
        </Dialog>
    );
}
