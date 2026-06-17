"use client";

import { useMemo } from "react";
import { CheckCircle, AlertTriangle, AlertCircle } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import type { EvalCompliance, WeightingIssue, AbsenteeRow } from "./types";

// ── 2D-1: CBC Evaluation Compliance ──
function EvalComplianceList({ items }: { items: EvalCompliance[] }) {
    const sorted = useMemo(() => {
        return [...items]
            .filter((i) => i.total > 0)
            .sort((a, b) => {
                const aPct = a.graded / a.total;
                const bPct = b.graded / b.total;
                if (aPct < 1 && bPct === 1) return -1;
                if (aPct === 1 && bPct < 1) return 1;
                return aPct - bPct;
            });
    }, [items]);

    return (
        <div>
            <span className="mb-2 block text-[11px] font-medium">CBC Evaluation Compliance</span>
            {sorted.length === 0 ? (
                <p className="text-muted-foreground text-[11px]">No tasks set</p>
            ) : (
                <div className="flex flex-col gap-2">
                    {sorted.map((row) => {
                        const pct = Math.round((row.graded / row.total) * 100);
                        const colorClass =
                            pct >= 100
                                ? "[&>div]:bg-emerald-500"
                                : pct >= 50
                                  ? ""
                                  : "[&>div]:bg-red-500";
                        return (
                            <div key={row.teacher} className="flex flex-col gap-0.5">
                                <div className="flex items-center justify-between text-[11px]">
                                    <span>
                                        <span className="font-medium">{row.teacher}</span>
                                        <span className="text-muted-foreground ml-1">
                                            {row.class}
                                        </span>
                                    </span>
                                    <span className="text-muted-foreground">
                                        {row.graded} / {row.total} graded
                                    </span>
                                </div>
                                <Progress value={pct} className={`h-1.5 ${colorClass}`} />
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}

// ── 2D-2: Weighting Audit ──
function WeightingAudit({ issues }: { issues: WeightingIssue[] }) {
    if (issues.length === 0) {
        return (
            <div className="text-muted-foreground flex items-center gap-1 text-[11px]">
                <CheckCircle className="size-3 text-emerald-500" />
                All task weights correctly configured
            </div>
        );
    }

    return (
        <div className="flex flex-col gap-2">
            {issues.map((issue, i) => {
                const isError = issue.severity === "error";
                return (
                    <Alert
                        key={i}
                        variant={isError ? "destructive" : "default"}
                        className={!isError ? "border-amber-300" : ""}
                    >
                        <div className="flex items-center gap-2">
                            {isError ? (
                                <AlertCircle className="size-4 shrink-0" />
                            ) : (
                                <AlertTriangle className="size-4 shrink-0 text-amber-600" />
                            )}
                            <AlertDescription className="text-xs">
                                <span className="font-mono text-[10px]">{issue.class}</span>{" "}
                                {issue.issue}
                            </AlertDescription>
                        </div>
                    </Alert>
                );
            })}
        </div>
    );
}

// ── 2D-3: Chronic Absentee Watchlist ──
function AbsenteeWatchlist({ rows }: { rows: AbsenteeRow[] }) {
    if (rows.length === 0) {
        return (
            <p className="text-muted-foreground text-[11px]">
                No students exceed the absence threshold
            </p>
        );
    }

    return (
        <div className="flex flex-col gap-2">
            {rows.map((row) => {
                const overPct = Math.min((row.absentDays / row.threshold) * 100, 150);
                return (
                    <div key={row.name} className="flex flex-col gap-1">
                        <div className="flex items-center gap-1.5 text-xs">
                            <span className="font-medium">{row.name}</span>
                            <span className="bg-muted text-muted-foreground rounded px-1 py-0.5 text-[10px]">
                                {row.class}
                            </span>
                            <Badge variant="destructive" className="ml-auto text-[10px]">
                                {row.absentDays} days absent
                            </Badge>
                        </div>
                        <div className="flex items-center gap-1">
                            <span className="text-muted-foreground text-[10px]">
                                Threshold: {row.threshold} days
                            </span>
                            <Progress value={overPct} className="h-1 flex-1 [&>div]:bg-red-500" />
                        </div>
                    </div>
                );
            })}
        </div>
    );
}

// ── Combined Driver ──
export function TeacherAudits({
    evalCompliance,
    weightingIssues,
    absentees,
}: {
    evalCompliance: EvalCompliance[];
    weightingIssues: WeightingIssue[];
    absentees: AbsenteeRow[];
}) {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-xs">Teacher &amp; Assessment Audits</CardTitle>
            </CardHeader>
            <CardContent>
                <div className="flex flex-col gap-3">
                    <EvalComplianceList items={evalCompliance} />
                    <Separator />
                    <WeightingAudit issues={weightingIssues} />
                    <Separator />
                    <AbsenteeWatchlist rows={absentees} />
                </div>
            </CardContent>
        </Card>
    );
}
