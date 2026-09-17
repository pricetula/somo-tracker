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

export default function AssignSlotPage() {
    const router = useRouter();
    const params = useParams();
    const searchParams = useSearchParams();

    const id = params.id as string;
    const dayOfWeek = Number(searchParams.get("day") ?? 1);
    const timeSlotId = searchParams.get("slot") ?? "";
    const classId = searchParams.get("classId");
    const gradeId = searchParams.get("gradeId");

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
                    <AlertDialogHeader>
                        <AlertDialogTitle>Time table not selected</AlertDialogTitle>
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
                    <AlertDialogHeader>
                        <AlertDialogTitle>Day not selected</AlertDialogTitle>
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
                    <AlertDialogHeader>
                        <AlertDialogTitle>Time slot not selected</AlertDialogTitle>
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
                    <AlertDialogHeader>
                        <AlertDialogTitle>Class not selected</AlertDialogTitle>
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

    if (!gradeId) {
        return (
            <AlertDialog open>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Grade not selected</AlertDialogTitle>
                        <AlertDialogDescription>
                            To assign a timetable slot, please select a grade first.
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
                gradeId={gradeId ?? ""}
                onSuccess={() => router.push(`/timetable/${id}`)}
            />
        </div>
    );
}
