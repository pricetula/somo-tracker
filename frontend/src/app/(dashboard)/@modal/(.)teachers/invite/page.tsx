"use client";

import { TeacherInviteOrchestrator } from "@/features/teachers";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function TeacherInviteModalPage() {
    return (
        <Dialog open>
            <DialogContent className="max-h-[85vh] max-w-4xl overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Invite Teachers</DialogTitle>
                </DialogHeader>
                <TeacherInviteOrchestrator />
            </DialogContent>
        </Dialog>
    );
}
