/**
 * IncidentList — displays medical incidents with an option to log new ones.
 *
 * SCHOOL_ADMIN / NURSE: sees all incidents for the school.
 * Can also filter by student.
 *
 * Uses the shared DataTable component for paginated listing.
 */
"use client";

import { useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { format } from "date-fns";
import { Trash2, Plus } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import { listMedicalIncidents, deleteMedicalIncident } from "@/lib/api/health";
import type { MedicalIncident } from "@/lib/api/health";
import { getErrorMessage } from "@/lib/errors";
import { CreateIncidentDialog } from "./create-incident-dialog";

// ─── Props ────────────────────────────────────────────────────────────────

interface IncidentListProps {
    studentId?: string;
}

// ─── Delete cell ──────────────────────────────────────────────────────────

function DeleteCell({ incident }: { incident: MedicalIncident }) {
    const queryClient = useQueryClient();

    const handleDelete = useCallback(async () => {
        try {
            await deleteMedicalIncident(incident.id);
            await queryClient.invalidateQueries({ queryKey: ["health", "incidents"] });
            toast.success("Incident deleted.");
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    }, [incident.id, queryClient]);

    return (
        <AlertDialog>
            <AlertDialogTrigger asChild>
                <Button variant="ghost" size="icon" className="size-6">
                    <Trash2 className="text-muted-foreground size-3" />
                </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Delete Incident</AlertDialogTitle>
                    <AlertDialogDescription>
                        Are you sure you want to delete this incident record? This cannot be undone.
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction onClick={handleDelete}>Delete</AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<MedicalIncident>[] = [
    {
        id: "symptoms",
        header: "Symptoms / Description",
        cell: (row) => <span className="font-medium">{row.symptoms}</span>,
    },
    {
        id: "action_taken",
        header: "Action Taken",
        width: "200px",
        cell: (row) => (
            <Badge variant="secondary" className="text-xs">
                {row.action_taken.length > 50
                    ? row.action_taken.slice(0, 50) + "…"
                    : row.action_taken}
            </Badge>
        ),
    },
    {
        id: "timestamp",
        header: "Date & Time",
        width: "180px",
        cell: (row) => (
            <span className="text-muted-foreground text-xs">
                {format(new Date(row.incident_timestamp), "MMM d, yyyy, h:mm a")}
                {row.logged_by_name && ` — by ${row.logged_by_name}`}
            </span>
        ),
    },
    {
        id: "actions",
        header: "",
        width: "48px",
        align: "right",
        cell: (row) => <DeleteCell incident={row} />,
    },
];

// ─── Wrapper query fn ─────────────────────────────────────────────────────

function createIncidentQueryFn(studentId?: string) {
    return (params: { page?: number; limit?: number; search?: string }) =>
        listMedicalIncidents({
            ...(studentId ? { student_id: studentId } : {}),
            page: params.page,
            limit: params.limit,
        });
}

// ─── Component ────────────────────────────────────────────────────────────

export function IncidentList({ studentId }: IncidentListProps) {
    const [showCreate, setShowCreate] = useState(false);
    const incidentQueryFn = createIncidentQueryFn(studentId);

    return (
        <div className="space-y-4">
            <DataTable
                queryKey={["health", "incidents", "list", studentId]}
                queryFn={incidentQueryFn}
                columns={columns}
                getRowId={(row) => row.id}
                emptyState="No incidents recorded."
                noResultsState="No incidents match your search."
                renderToolBarComponents={() => (
                    <Button
                        key="log-incident"
                        variant="outline"
                        size="sm"
                        onClick={() => setShowCreate(true)}
                    >
                        <Plus className="mr-1 size-4" />
                        Log Incident
                    </Button>
                )}
            />

            <CreateIncidentDialog
                open={showCreate}
                onOpenChange={setShowCreate}
                studentId={studentId}
            />
        </div>
    );
}
