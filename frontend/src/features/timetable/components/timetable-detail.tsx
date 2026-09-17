"use client";

import React from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ClassesCombobox } from "@/features/classes/components/classes-combobox";
import { TimetableGrid } from "./timetable-grid";
import { useTimeSlots } from "../hooks/use-time-slots";
import { useTimetableTemplate } from "../hooks/use-timetable-template";
import { TemplateHeaderEdit } from "./template-header-edit";

type Props = {
    templateId: string;
};

export function TimetableDetail({ templateId }: Props) {
    const [selectedIds, setSelectedIds] = React.useState({ classId: "", gradeId: "" });
    const { data: slots = [], isLoading } = useTimeSlots(templateId);
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
                    onChange={(v, d) => {
                        setSelectedIds({
                            classId: v,
                            gradeId: d?.gradeId || "",
                        });
                    }}
                    placeholder="Select class"
                />
            </header>
            <TimetableGrid
                slots={slots}
                isLoading={isLoading}
                templateId={templateId}
                selectedIds={selectedIds}
            />
        </div>
    );
}
