"use client";

import { LayoutGrid } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface EmptyStateAllocationProps {
    onAssignLesson?: () => void;
    onManageStructure?: () => void;
}

export function EmptyStateAllocation({
    onAssignLesson,
    onManageStructure,
}: EmptyStateAllocationProps) {
    return (
        <div className="flex h-[60vh] items-center justify-center">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <LayoutGrid className="text-muted-foreground mx-auto h-12 w-12" />
                    <CardTitle className="mt-4">Structure Exists, No Lessons Assigned</CardTitle>
                    <p className="text-muted-foreground">
                        Your weekly schedule is defined, but no classes have been assigned to
                        periods yet. Click a cell to assign a lesson.
                    </p>
                </CardHeader>
                <CardContent className="space-y-3">
                    <Button
                        onClick={onAssignLesson}
                        size="lg"
                        className="w-full"
                        disabled={!onAssignLesson}
                    >
                        Assign First Lesson
                    </Button>
                    <Button
                        variant="outline"
                        onClick={onManageStructure}
                        className="w-full"
                        disabled={!onManageStructure}
                    >
                        Manage Time Blocks
                    </Button>
                </CardContent>
            </Card>
        </div>
    );
}
