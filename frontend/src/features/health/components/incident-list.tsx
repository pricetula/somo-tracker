"use client";

import { useState } from "react";
import { format } from "date-fns";
import { Plus } from "lucide-react";
import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { listMedicalIncidents, deleteMedicalIncident } from "@/lib/api/health";
import { type MedicalIncident } from "@/lib/api/health";
import { CreateIncidentDialog } from "./create-incident-dialog";

interface IncidentListProps {
    studentId?: string;
}
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
function createIncidentQueryFn(studentId?: string) {
    return (params: { page?: number; limit?: number; search?: string }) =>
        listMedicalIncidents({
            ...(studentId ? { student_id: studentId } : {}),
            page: params.page,
            limit: params.limit,
        });
}

import { DeleteCell } from "./delete-cell";

export function IncidentList({ studentId }: IncidentListProps) {
    const [showCreate, setShowCreate] = useState(false);
    const incidentQueryFn = createIncidentQueryFn(studentId);

    return (
        <div className="space-y-4">
            <DataTable
                isCheckable
                queryKey={["health", "incidents", "list", studentId]}
                queryFn={incidentQueryFn}
                columns={columns}
                getRowId={(row) => row.id}
                deleteFn={(id) => deleteMedicalIncident(String(id))}
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
