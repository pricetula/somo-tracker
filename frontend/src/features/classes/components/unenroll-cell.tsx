"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { UserMinus } from "lucide-react";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { type RowAction } from "@/components/shared/data-table/row-actions";
import { unenrollStudent, type RosterEntry } from "@/lib/api/classes";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export function UnenrollCell({
    classId,
    student,
    academicTermId,
}: {
    classId: string;
    student: RosterEntry;
    academicTermId?: string;
}) {
    const queryClient = useQueryClient();

    const unenrollMutation = useMutation({
        mutationFn: () => unenrollStudent(classId, student.id, academicTermId),
        onSuccess: () => {
            toast.success(`${student.full_name} unenrolled.`);
            queryClient.invalidateQueries({ queryKey: ["class-roster", classId] });
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });

    const rowActions: RowAction[] = [
        {
            label: "Unenroll",
            icon: UserMinus,
            destructive: true,
            confirmTitle: "Unenroll Student",
            confirmDescription: `Are you sure you want to unenroll "${student.full_name}" from this class? Their enrollment record will be marked as suspended.`,
            onClick: () => unenrollMutation.mutate(),
        },
    ];

    return (
        <RowActions
            rowId={student.id}
            label={student.full_name}
            actions={rowActions}
            disabled={unenrollMutation.isPending}
        />
    );
}
