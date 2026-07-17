/**
 * GradingScaleProfilesList — Admin view listing all grading scale profiles.
 *
 * Uses the shared DataTable component with toggle-active and delete actions.
 */

"use client";

import Link from "next/link";
import { ToggleLeft, ToggleRight, Trash2 } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

import type { ScaleProfile } from "@/lib/api/assessments";
import { listScaleProfiles } from "@/lib/api/assessments";
import { useToggleScaleProfile, useDeleteScaleProfile } from "../hooks/use-assessments";
import { getErrorMessage } from "@/lib/errors";

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
        <AlertDialog>
            <AlertDialogTrigger asChild>
                <Button
                    variant="ghost"
                    size="icon-sm"
                    className="text-muted-foreground hover:text-destructive"
                >
                    <Trash2 className="h-4 w-4" />
                    <span className="sr-only">Delete {profileName}</span>
                </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Delete Scale Profile</AlertDialogTitle>
                    <AlertDialogDescription>
                        Are you sure you want to delete <strong>{profileName}</strong>? This will
                        also remove all its percentage ranges.
                        {deleteMutation.isError && (
                            <p className="text-destructive mt-2">
                                {getErrorMessage(deleteMutation.error)}
                            </p>
                        )}
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                        onClick={(e) => {
                            e.preventDefault();
                            deleteMutation.mutate(profileId);
                        }}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    >
                        {deleteMutation.isPending ? "Deleting..." : "Delete"}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
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
        id: "delete",
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

    return (
        <DataTable
            addHref="/assessments/grading-scales/new"
            queryKey={["scale-profiles"]}
            queryFn={profilesQueryFn}
            columns={columns}
            getRowId={(row) => row.id}
            emptyState="No grading scale profiles yet. Create one to define how percentages map to CBC levels."
            noResultsState="No profiles match your search."
        />
    );
}
