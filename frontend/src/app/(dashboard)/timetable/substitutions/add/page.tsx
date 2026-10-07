"use client";

import { useRouter } from "next/navigation";
import { SubstitutionForm } from "@/features/timetable";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function SubstitutionAddModalPage() {
    const router = useRouter();

    const handleOpenChange = (open: boolean) => {
        if (!open) {
            router.back();
        }
    };

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] overflow-y-auto md:max-w-xl">
                <DialogHeader>
                    <DialogTitle>Add Substitution</DialogTitle>
                </DialogHeader>
                <SubstitutionForm onSuccess={() => router.push("/timetable/substitutions")} />
            </DialogContent>
        </Dialog>
    );
}
