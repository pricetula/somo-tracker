/**
 * Intercepted route for creating a new student via the modal slot.
 *
 * Renders the StudentForm inside a Dialog when navigating to /students/new
 * from within the dashboard, preserving the background page.
 */

"use client";

import { useRouter } from "next/navigation";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { StudentForm } from "@/features/students";

export default function NewStudentModal() {
    const router = useRouter();

    const handleSuccess = (id: string) => {
        router.push(`/students/${id}`);
    };

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent className="sm:max-w-xl">
                <DialogHeader>
                    <DialogTitle>Add New Student</DialogTitle>
                    <DialogDescription>
                        Enter the student&apos;s demographic information. You can enroll them in a
                        class after creating their profile.
                    </DialogDescription>
                </DialogHeader>
                <StudentForm mode="create" onSuccess={handleSuccess} />
            </DialogContent>
        </Dialog>
    );
}
