"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useGrades } from "@/features/grades/hooks/use-grades";
import { useStreams } from "@/features/streams/hooks/use-streams";
import { createClass } from "@/features/classes/services/api";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

interface CreateClassesBulkProps {
    onSuccess: () => void;
}

export function CreateClassesBulk({ onSuccess }: CreateClassesBulkProps) {
    const { data: grades = [], isLoading: gradesLoading } = useGrades();
    const { data: streams = [], isLoading: streamsLoading } = useStreams();
    const [isCreating, setIsCreating] = React.useState(false);
    const [createdCount, setCreatedCount] = React.useState(0);

    const totalCombinations = grades.length * streams.length;

    const handleCreate = async () => {
        if (grades.length === 0 || streams.length === 0) {
            toast.error("Grades or streams are missing. Complete previous steps first.");
            return;
        }
        setIsCreating(true);
        setCreatedCount(0);
        let success = 0;
        let failed = 0;

        for (const grade of grades) {
            for (const stream of streams) {
                try {
                    const name = `${grade.local_label} ${stream.name}`;
                    await createClass({
                        name,
                        gradeId: grade.id,
                        streamId: stream.id,
                    });
                    success++;
                    setCreatedCount(success);
                } catch (e) {
                    failed++;
                    console.error("Failed to create class", grade.id, stream.id, e);
                }
            }
        }

        setIsCreating(false);
        if (failed === 0) {
            toast.success(`Created ${success} classes for all grades and streams`);
            onSuccess();
        } else {
            toast.error(`Created ${success} classes, ${failed} failed`);
            onSuccess();
        }
    };

    return (
        <div className="space-y-6">
            <h2 className="text-xl font-semibold">Bulk Create Classes</h2>
            <p className="text-muted-foreground text-sm">
                This will create one class for every grade × stream combination using the current
                academic year.
            </p>

            <div className="space-y-4">
                <div>
                    <h3 className="mb-2 font-medium">Grades ({grades.length})</h3>
                    <div className="flex flex-wrap gap-2">
                        {gradesLoading && (
                            <span className="text-muted-foreground text-sm">Loading grades…</span>
                        )}
                        {grades.map((g) => (
                            <Badge key={g.id} variant="secondary">
                                {g.local_label}
                            </Badge>
                        ))}
                    </div>
                </div>

                <div>
                    <h3 className="mb-2 font-medium">Streams ({streams.length})</h3>
                    <div className="flex flex-wrap gap-2">
                        {streamsLoading && (
                            <span className="text-muted-foreground text-sm">Loading streams…</span>
                        )}
                        {streams.map((s) => (
                            <Badge key={s.id} variant="secondary">
                                {s.name}
                            </Badge>
                        ))}
                    </div>
                </div>
            </div>

            <div className="bg-muted rounded-md p-3 text-sm">
                Total classes to create: <span className="font-medium">{totalCombinations}</span>
                {isCreating && (
                    <span className="text-muted-foreground ml-2">
                        Created {createdCount}/{totalCombinations}
                    </span>
                )}
            </div>

            <div className="pt-2">
                <Button
                    onClick={handleCreate}
                    disabled={
                        isCreating || gradesLoading || streamsLoading || totalCombinations === 0
                    }
                >
                    {isCreating ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Creating...
                        </>
                    ) : (
                        `Create ${totalCombinations} Classes`
                    )}
                </Button>
            </div>
        </div>
    );
}
