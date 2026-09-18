"use client";

import { useRouter, useParams, useSearchParams } from "next/navigation";
import { AssignSlotForm } from "@/features/timetable/components/assign-slot-form";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { AlertCircle } from "lucide-react";

export default function AssignSlotPage() {
    const router = useRouter();
    const params = useParams();
    const searchParams = useSearchParams();

    const id = params.id as string;
    const dayOfWeek = Number(searchParams.get("day") ?? 1);
    const timeSlotId = searchParams.get("slot") ?? "";
    const classId = searchParams.get("classId");

    const handleContinue = () => {
        let url = "/timetable";
        if (id) {
            url += `/${id}`;
        }
        window.location.href = url;
    };

    if (!id) {
        return (
            <AlertDialog open>
                <AlertDialogContent>
                    <AlertDialogHeader className="mb-2">
                        <AlertDialogTitle className="text-destructive flex items-center gap-2">
                            <AlertCircle size="16" />
                            <span>Time table not selected</span>
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                            To continue, please select a time table first.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogAction onClick={handleContinue}>Continue</AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        );
    }

    if (!dayOfWeek) {
        return (
            <AlertDialog open>
                <AlertDialogContent>
                    <AlertDialogHeader className="mb-2">
                        <AlertDialogTitle className="text-destructive flex items-center gap-2">
                            <AlertCircle size="16" />
                            <span>Day not selected</span>
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                            To continue, please select a day first.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogAction onClick={handleContinue}>Continue</AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        );
    }

    if (!timeSlotId) {
        return (
            <AlertDialog open>
                <AlertDialogContent>
                    <AlertDialogHeader className="mb-2">
                        <AlertDialogTitle className="text-destructive flex items-center gap-2">
                            <AlertCircle size="16" />
                            <span>Time slot not selected</span>
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                            To continue, please select a time slot first.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogAction onClick={handleContinue}>Continue</AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        );
    }

    if (!classId) {
        return (
            <AlertDialog open>
                <AlertDialogContent>
                    <AlertDialogHeader className="mb-2">
                        <AlertDialogTitle className="text-destructive flex items-center gap-2">
                            <AlertCircle size="16" />
                            <span>Class not selected</span>
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                            To assign a timetable slot, please select a class first.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogAction onClick={handleContinue}>Continue</AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        );
    }

    return (
        <div className="mx-auto max-w-2xl space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Assign Timetable Slot</h1>
            <AssignSlotForm
                dayOfWeek={dayOfWeek}
                timeSlotId={timeSlotId}
                classId={classId ?? ""}
                onSuccess={() => router.push(`/timetable/${id}`)}
            />
        </div>
    );
}
