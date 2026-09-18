"use client";

import React from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ClassesCombobox } from "@/features/classes/components/classes-combobox";
import { TimetableGrid } from "./timetable-grid";
import { useTimeSlots } from "../hooks/use-time-slots";
import { useTimetableTemplate } from "../hooks/use-timetable-template";
import { useClassTimetableSlots } from "../hooks/use-class-timetable-slots";
import { TemplateHeaderEdit } from "./template-header-edit";

type Props = {
    templateId: string;
};

export function TimetableDetail({ templateId }: Props) {
    const router = useRouter();
    const searchParams = useSearchParams();
    const classId = searchParams.get("classId") || "";
    const selectedIds = React.useMemo(() => ({ classId, gradeId: "" }), [classId]);

    const { data: slots = [], isLoading } = useTimeSlots(templateId);
    const { data: assignments = [], isLoading: assignmentsLoading } = useClassTimetableSlots(
        templateId,
        selectedIds.classId
    );
    const {
        data: template,
        isLoading: templateLoading,
        isError,
    } = useTimetableTemplate(templateId);

    if (isError) {
        return (
            <div className="p-6">
                <Alert variant="destructive">
                    <AlertDescription>Failed to load template details.</AlertDescription>
                </Alert>
            </div>
        );
    }

    return (
        <div className="space-y-6 p-6">
            <header className="flex items-center justify-between">
                <TemplateHeaderEdit
                    templateId={templateId}
                    initialName={template?.name ?? ""}
                    initialDescription={template?.description ?? ""}
                    loading={templateLoading}
                />
                <ClassesCombobox
                    value={selectedIds.classId}
                    onChange={(v) => {
                        const params = new URLSearchParams(searchParams.toString());
                        if (v) {
                            params.set("classId", v);
                        } else {
                            params.delete("classId");
                        }
                        router.replace(`?${params.toString()}`, { scroll: false });
                    }}
                    placeholder="Select class"
                />
            </header>
            <TimetableGrid
                slots={slots}
                isLoading={isLoading || assignmentsLoading}
                templateId={templateId}
                selectedIds={selectedIds}
                assignments={assignments}
            />
        </div>
    );
}
