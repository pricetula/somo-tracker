"use client";

import { BookOpen } from "lucide-react";
import { type StudentTermGrade } from "@/lib/api/assessments";
import { PerformanceLevelBadge } from "./performance-level-badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function TermGradeCard({ grade }: { grade: StudentTermGrade }) {
    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                    <div>
                        <CardTitle className="text-sm font-semibold">
                            {grade.learning_area_name}
                        </CardTitle>
                        <CardDescription className="text-xs">
                            {grade.learning_area_code}
                        </CardDescription>
                    </div>
                    <PerformanceLevelBadge level={grade.final_level} showLabel />
                </div>
            </CardHeader>
            <CardContent>
                <div className="text-muted-foreground flex items-center gap-1.5 text-xs">
                    <BookOpen className="h-3 w-3" />
                    <span>
                        Based on <strong>{grade.assessment_count}</strong> assessment
                        {grade.assessment_count !== 1 ? "s" : ""}
                    </span>
                </div>
            </CardContent>
        </Card>
    );
}
