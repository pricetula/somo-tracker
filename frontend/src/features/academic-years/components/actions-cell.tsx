"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { setCurrentYear, deleteAcademicYear } from "@/lib/api/academic-terms";
import { type AcademicYear } from "@/lib/api/academic-terms";
import { getErrorMessage } from "@/lib/errors";
import Link from "next/link";

export function ActionsCell({ year }: { year: AcademicYear }) {
    const queryClient = useQueryClient();

    const setCurrentMutation = useMutation({
        mutationFn: () => setCurrentYear(year.id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["academic-years"] });
            toast.success(`${year.name} set as current year.`);
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });

    const deleteMutation = useMutation({
        mutationFn: () => deleteAcademicYear(year.id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["academic-years"] });
            toast.success("Academic year deleted.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });

    return (
        <div className="flex items-center justify-end gap-2">
            {!year.is_current && (
                <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCurrentMutation.mutate()}
                    disabled={setCurrentMutation.isPending}
                >
                    Set Current
                </Button>
            )}
            <Button variant="outline" size="sm" asChild>
                <Link href={`/academic-years/${year.id}`}>Edit</Link>
            </Button>
            <RowActions
                rowId={year.id}
                label={year.name}
                onDelete={() => deleteMutation.mutate()}
                disabled={deleteMutation.isPending}
            />
        </div>
    );
}
