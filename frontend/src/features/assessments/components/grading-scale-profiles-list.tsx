"use client";

import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { type ScaleProfile } from "@/lib/api/assessments";
import { listScaleProfiles } from "@/lib/api/assessments";
import { useDeleteScaleProfile } from "../hooks/use-assessments";
import Link from "next/link";

const columns: DataTableColumn<ScaleProfile>[] = [
    {
        id: "name",
        header: "Profile Name",
        cell: (row) => (
            <Link
                href={`/assessments/grading-scales/${row.id}`}
                className="font-medium hover:underline"
            >
                {row.name}
            </Link>
        ),
    },
    {
        id: "is_active",
        header: "Status",
        width: "100px",
        cell: (row) => (
            <Badge
                variant="secondary"
                className={
                    row.is_active
                        ? "bg-emerald-100 text-emerald-700"
                        : "bg-muted text-muted-foreground"
                }
            >
                {row.is_active ? "Active" : "Inactive"}
            </Badge>
        ),
    },
    {
        id: "toggle",
        header: "",
        width: "48px",
        align: "right",
        cell: (row) => <ActiveToggle profile={row} />,
    },
    {
        id: "actions",
        header: "",
        width: "48px",
        align: "right",
        cell: (row) => <DeleteCell profileId={row.id} profileName={row.name} />,
    },
];
function createProfilesQueryFn() {
    return (_params: { page?: number; limit?: number }) => listScaleProfiles();
}

import { ActiveToggle } from "./active-toggle";
import { DeleteCell } from "./delete-cell";

export function GradingScaleProfilesList() {
    const profilesQueryFn = createProfilesQueryFn();

    const deleteMutation = useDeleteScaleProfile();

    return (
        <DataTable
            isCheckable
            addHref="/assessments/grading-scales/new"
            queryKey={["scale-profiles"]}
            queryFn={profilesQueryFn}
            columns={columns}
            getRowId={(row) => row.id}
            deleteFn={(id) => deleteMutation.mutateAsync(String(id)).then(() => {})}
            emptyState="No grading scale profiles yet. Create one to define how percentages map to CBC levels."
            noResultsState="No profiles match your search."
        />
    );
}
