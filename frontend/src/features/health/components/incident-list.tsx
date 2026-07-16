/**
 * IncidentList — displays medical incidents with an option to log new ones.
 *
 * SCHOOL_ADMIN / NURSE: sees all incidents for the school.
 * Can also filter by student.
 */
"use client";

import { useState } from "react";
import { useMedicalIncidents, useDeleteMedicalIncident } from "../hooks/use-health";
import { CreateIncidentDialog } from "./create-incident-dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Plus, Trash2 } from "lucide-react";

export function IncidentList({ studentId }: { studentId?: string }) {
    const [showCreate, setShowCreate] = useState(false);
    const { data, isLoading } = useMedicalIncidents(
        studentId ? { student_id: studentId } : undefined
    );
    const deleteMutation = useDeleteMedicalIncident();

    if (isLoading) {
        return <p className="text-muted-foreground">Loading incidents…</p>;
    }

    const incidents = data?.items ?? [];

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h3 className="text-foreground font-medium">Medical Incidents</h3>
                <Button variant="outline" size="sm" onClick={() => setShowCreate(true)}>
                    <Plus className="mr-1 size-4" />
                    Log Incident
                </Button>
            </div>

            {incidents.length === 0 ? (
                <p className="text-muted-foreground">No incidents recorded.</p>
            ) : (
                <div className="space-y-3">
                    {incidents.map((inc) => (
                        <div key={inc.id} className="bg-muted/30 space-y-1 rounded-md p-3">
                            <div className="flex items-start justify-between gap-2">
                                <div className="space-y-1">
                                    <p className="text-foreground font-medium">{inc.symptoms}</p>
                                    <p className="text-muted-foreground text-xs">
                                        {new Date(inc.incident_timestamp).toLocaleString()}
                                        {inc.logged_by_name && ` — by ${inc.logged_by_name}`}
                                    </p>
                                </div>
                                <div className="flex shrink-0 items-center gap-2">
                                    <Badge variant="secondary" className="text-xs">
                                        {inc.action_taken.length > 40
                                            ? inc.action_taken.slice(0, 40) + "…"
                                            : inc.action_taken}
                                    </Badge>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        className="size-6"
                                        onClick={() => deleteMutation.mutate(inc.id)}
                                    >
                                        <Trash2 className="text-muted-foreground size-3" />
                                    </Button>
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}

            <CreateIncidentDialog
                open={showCreate}
                onOpenChange={setShowCreate}
                studentId={studentId}
            />
        </div>
    );
}
