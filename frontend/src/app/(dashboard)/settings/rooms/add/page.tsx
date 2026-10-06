"use client";

import { useRouter } from "next/navigation";
import { RoomForm } from "@/features/rooms";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function RoomAddModalPage() {
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
                    <DialogTitle>Add Room</DialogTitle>
                </DialogHeader>
                <RoomForm onSuccess={() => router.push("/school/rooms")} />
            </DialogContent>
        </Dialog>
    );
}
