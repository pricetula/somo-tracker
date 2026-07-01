/**
 * Sessions listing page.
 *
 * Shows all assessment sessions for the school, filterable by class and blueprint.
 * Maps to GET /api/v1/assessment/sessions.
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

import { SessionsTable, useSessions, useDeleteSession } from "@/features/assessment";

export default function SessionsPage() {
    const router = useRouter();

    const {
        data: sessionsData,
        isLoading: sessionsLoading,
        isError: sessionsError,
    } = useSessions();

    const deleteSession = useDeleteSession();

    const sessions = sessionsData?.data ?? [];

    const handleDelete = async (id: string) => {
        if (window.confirm("Delete this session? This will also remove all recorded scores.")) {
            deleteSession.mutate(id);
        }
    };

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Assessment Sessions</h1>
                <div className="ml-auto flex items-center gap-2">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => router.push("/assessment/sessions/new")}
                    >
                        <Plus className="mr-1.5 size-3.5" />
                        New Session
                    </Button>
                </div>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-1 flex-col">
                    {sessionsError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load sessions. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <SessionsTable
                                sessions={sessions}
                                total={sessions.length}
                                isLoading={sessionsLoading}
                                onDelete={handleDelete}
                                onCreateClick={() => router.push("/assessment/sessions/new")}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
