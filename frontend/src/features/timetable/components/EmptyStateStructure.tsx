"use client";

import { CalendarPlus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface EmptyStateStructureProps {
    onCreateStructure?: () => void;
}

export function EmptyStateStructure({ onCreateStructure }: EmptyStateStructureProps) {
    return (
        <div className="flex h-[60vh] items-center justify-center">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <CalendarPlus className="text-muted-foreground mx-auto h-12 w-12" />
                    <CardTitle className="mt-4">No Timetable Structure</CardTitle>
                    <p className="text-muted-foreground">
                        You haven&apos;t defined the weekly schedule yet. Create time blocks
                        (periods, breaks) to establish the grid structure.
                    </p>
                </CardHeader>
                <CardContent className="text-center">
                    <Button onClick={onCreateStructure} size="lg" disabled={!onCreateStructure}>
                        Create Timetable Structure
                    </Button>
                </CardContent>
            </Card>
        </div>
    );
}
