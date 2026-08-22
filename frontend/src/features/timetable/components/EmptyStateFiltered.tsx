"use client";

import { FilterX, PlusCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface EmptyStateFilteredProps {
    filterLabel: string;
    onClearFilters?: () => void;
    onAddLesson?: () => void;
}

export function EmptyStateFiltered({
    filterLabel,
    onClearFilters,
    onAddLesson,
}: EmptyStateFilteredProps) {
    return (
        <div className="flex h-[60vh] items-center justify-center">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <FilterX className="text-muted-foreground mx-auto h-12 w-12" />
                    <CardTitle className="mt-4">No Lessons Found</CardTitle>
                    <p className="text-muted-foreground">
                        No lessons match your current filter:{" "}
                        <span className="font-medium">{filterLabel}</span>
                    </p>
                </CardHeader>
                <CardContent className="space-y-3">
                    {onAddLesson && (
                        <Button onClick={onAddLesson} size="lg" className="w-full">
                            <PlusCircle className="mr-2 h-4 w-4" />
                            Add Lesson for This Filter
                        </Button>
                    )}
                    <Button
                        variant="outline"
                        onClick={onClearFilters}
                        className="w-full"
                        disabled={!onClearFilters}
                    >
                        Clear Filters
                    </Button>
                </CardContent>
            </Card>
        </div>
    );
}
