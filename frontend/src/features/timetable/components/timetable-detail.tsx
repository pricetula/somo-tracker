"use client";

import React from "react";
import { ClassesCombobox } from "@/features/classes/components/classes-combobox";
import { TimetableGrid } from "./timetable-grid";
import { useTimeSlots } from "../hooks/use-time-slots";

type Props = {
    templateId: string;
};

export function TimetableDetail({ templateId }: Props) {
    const [selectedIds, setSelectedIds] = React.useState({ classId: "", gradeId: "" });
    const { data: slots = [], isLoading } = useTimeSlots(templateId);

    return (
        <div className="space-y-6 p-6">
            <header>
                <h1 className="text-2xl font-semibold">Timetable Template</h1>
                <ClassesCombobox
                    value={selectedIds.classId}
                    onChange={(v, d) => {
                        console.log(";dddd", d);
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
