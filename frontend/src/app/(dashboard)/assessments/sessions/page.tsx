"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { listSessions, type AssessmentSession } from "@/lib/api/assessments";
import { Button } from "@/components/ui/button";

export default function SessionsListPage() {
    const { data, isLoading } = useQuery({
        queryKey: ["assessments", "sessions"],
        queryFn: () => listSessions({ page: 1, limit: 50 }),
    });

    return (
        <div className="p-6">
            <div className="mb-6 flex items-center justify-between">
                <h1 className="text-xl font-semibold">Assessment Sessions</h1>
                <Button asChild>
                    <Link href="/assessments/sessions/new">New Session</Link>
                </Button>
            </div>

            {isLoading ? (
                <p className="text-muted-foreground text-sm">Loading...</p>
            ) : !data || data.items.length === 0 ? (
                <p className="text-muted-foreground text-sm">No sessions yet.</p>
            ) : (
                <div className="space-y-2">
                    {data.items.map((s: AssessmentSession) => (
                        <Link
                            key={s.id}
                            href={`/assessments/sessions/${s.id}`}
                            className="border-border hover:bg-muted block rounded border p-3"
                        >
                            <div className="flex items-center justify-between">
                                <div>
                                    <div className="font-medium">{s.name}</div>
                                    <div className="text-muted-foreground text-xs">
                                        {s.evaluation_method} · {s.status}
                                    </div>
                                </div>
                                <div className="text-muted-foreground text-xs">
                                    {new Date(s.created_at).toLocaleDateString()}
                                </div>
                            </div>
                        </Link>
                    ))}
                </div>
            )}
        </div>
    );
}
