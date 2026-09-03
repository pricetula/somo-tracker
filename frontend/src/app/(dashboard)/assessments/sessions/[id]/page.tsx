"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    getSession,
    listScores,
    upsertScores,
    listRubricOutcomes,
    upsertRubricOutcomes,
} from "@/lib/api/assessments";
import { QuantitativeSheet } from "@/features/assessments/components/quantitative-sheet";
import { RubricSheet } from "@/features/assessments/components/rubric-sheet";

interface Props {
    params: Promise<{ id: string }>;
}

export default function AssessmentDetailPage({ params }: Props) {
    // params is a Promise in this app's Next.js version; resolve via React.use()
    // (using a state-based read for simplicity).
    const sessionId = (params as unknown as { id: string })?.id ?? "";

    const { data: session } = useQuery({
        queryKey: ["assessments", "session", sessionId],
        queryFn: () => getSession(sessionId),
        enabled: !!sessionId,
    });

    const { data: scores } = useQuery({
        queryKey: ["assessments", "scores", sessionId],
        queryFn: () => listScores(sessionId, 1, 200),
        enabled: !!sessionId,
    });

    const { data: rubric } = useQuery({
        queryKey: ["assessments", "rubric", sessionId],
        queryFn: () => listRubricOutcomes(sessionId),
        enabled: !!sessionId,
    });

    const queryClient = useQueryClient();
    const saveMutation = useMutation({
        mutationFn: (entries: { student_id: string; raw_score: number | null }[]) =>
            upsertScores(sessionId, entries),
        onSuccess: () =>
            queryClient.invalidateQueries({ queryKey: ["assessments", "scores", sessionId] }),
    });

    const rubricMutation = useMutation({
        mutationFn: (
            entries: {
                student_id: string;
                performance_indicator_id: string;
                awarded_level: string;
            }[]
        ) => upsertRubricOutcomes(sessionId, entries),
        onSuccess: () =>
            queryClient.invalidateQueries({ queryKey: ["assessments", "rubric", sessionId] }),
    });

    const rows = (scores?.items ?? []).map((s) => ({
        student_id: s.student_id,
        student_name: s.student_id.slice(0, 8),
        raw_score: s.raw_score,
    }));

    return (
        <div className="space-y-6 p-6">
            <h1 className="text-2xl font-semibold">{session?.name ?? "Assessment"}</h1>
            <p className="text-muted-foreground text-sm">
                Status: {session?.status} · Method: {session?.evaluation_method}
                {session?.max_points != null ? ` · Max: ${session.max_points}` : ""}
            </p>

            {session?.evaluation_method === "QUANTITATIVE" ? (
                <QuantitativeSheet
                    maxPoints={session.max_points ?? 100}
                    rows={rows}
                    readOnly={session.status === "PUBLISHED"}
                    onSave={async (entries) => {
                        await saveMutation.mutateAsync(entries);
                    }}
                />
            ) : session?.evaluation_method === "RUBRIC" ? (
                <RubricSheet
                    rows={(rubric?.items ?? []).map((r) => ({
                        student_id: r.student_id,
                        student_name: r.student_id.slice(0, 8),
                        indicator_id: r.performance_indicator_id,
                        awarded_level: r.awarded_level as "EE" | "ME" | "AE" | "BE",
                    }))}
                    readOnly={session.status === "PUBLISHED"}
                    onSave={async (entries) => {
                        await rubricMutation.mutateAsync(entries);
                    }}
                />
            ) : null}
        </div>
    );
}
