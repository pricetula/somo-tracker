"use client";

import { User } from "lucide-react";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";

export function ChildSelector({
    studentList,
    selectedId,
    onChange,
}: {
    studentList: { student_id: string; full_name: string }[];
    selectedId: string;
    onChange: (id: string) => void;
}) {
    if (studentList.length <= 1) return null;

    return (
        <div className="flex items-center gap-2">
            <User className="text-muted-foreground h-4 w-4" />
            <Select value={selectedId} onValueChange={onChange}>
                <SelectTrigger className="w-64">
                    <SelectValue placeholder="Select a child..." />
                </SelectTrigger>
                <SelectContent>
                    {studentList.map((child) => (
                        <SelectItem key={child.student_id} value={child.student_id}>
                            {child.full_name}
                        </SelectItem>
                    ))}
                </SelectContent>
            </Select>
        </div>
    );
}
