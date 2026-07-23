/**
 * GradingScaleProfilesList — Admin view listing all grading scale profiles.
 *
 * Uses the shared DataTable component with toggle-active and delete actions.
 */

"use client";

import Link from "next/link";
import { ToggleLeft, ToggleRight } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

import type { ScaleProfile } from "@/lib/api/assessments";
import { listScaleProfiles } from "@/lib/api/assessments";
import { useToggleScaleProfile, useDeleteScaleProfile } from "../hooks/use-assessments";

// ─── Columns ──────────────────────────────────────────────────────────────

function ActiveToggle({ profile }: { profile: ScaleProfile }) {
    const toggleMutation = useToggleScaleProfile();

    return (
        <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => toggleMutation.mutate({ id: profile.id, isActive: !profile.is_active })}
            disabled={toggleMutation.isPending}
            title={profile.is_active ? "Deactivate" : "Activate"}
        >
            {profile.is_active ? (
                <ToggleRight className="h-4 w-4 text-emerald-600" />
            ) : (
                <ToggleLeft className="text-muted-foreground h-4 w-4" />
            )}
        </Button>
    );
}

function DeleteCell({ profileId, profileName }: { profileId: string; profileName: string }) {
    const deleteMutation = useDeleteScaleProfile();

    return (
        <RowActions
            rowId={profileId}
            label={profileName}
            onDelete={() => deleteMutation.mutate(profileId)}
            disabled={deleteMutation.isPending}
        />
    );
}

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

// ─── Wrapper query function ───────────────────────────────────────────────

/**
 * Wraps listScaleProfiles into the ListApiFn signature expected by DataTable.
 * Scale profiles have no server-side pagination — we fetch all and let the
 * DataTable handle client-side search.
 */
function createProfilesQueryFn() {
    return (_params: { page?: number; limit?: number }) => listScaleProfiles();
}

// ─── Component ────────────────────────────────────────────────────────────

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
