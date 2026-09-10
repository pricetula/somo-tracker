"use client";

import React, { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Plus, Trash2 } from "lucide-react";
import { useCreateStreams, useGrades } from "../hooks/use-streams";
import { useMeSession } from "@/features/auth/hooks/use-me-session";

interface CreateStreamsProps {
    onSuccess: () => void;
}

export function CreateStreams({ onSuccess }: CreateStreamsProps) {
    const { data: me } = useMeSession();
    const { data: gradesData } = useGrades();
    const { mutate: createStreams, isPending } = useCreateStreams();

    const [inputValue, setInputValue] = useState("");
    const [names, setNames] = useState<string[]>([]);

    const grades = gradesData?.grades ?? [];
    const firstGrade = grades[0];

    const preview = useMemo(() => {
        if (names.length === 0 || !firstGrade) return "";
        const gradeLabel = firstGrade.local_label ?? "Grade 1";
        return `${gradeLabel} ${names[0]}`;
    }, [names, firstGrade]);

    const addName = () => {
        const trimmed = inputValue.trim();
        if (!trimmed) return;
        if (!names.includes(trimmed)) {
            setNames((prev) => [...prev, trimmed]);
        }
        setInputValue("");
    };

    const removeName = (n: string) => {
        setNames((prev) => prev.filter((x) => x !== n));
    };

    const handleSubmit = () => {
        if (names.length === 0) return;
        createStreams(names, {
            onSuccess,
        });
    };

    return (
        <div className="space-y-6">
            <h2 className="text-xl font-semibold">Create Streams</h2>

            <div className="flex gap-2">
                <Input
                    value={inputValue}
                    onChange={(e) => setInputValue(e.target.value)}
                    onKeyDown={(e) => {
                        if (e.key === "Enter") {
                            e.preventDefault();
                            addName();
                        }
                    }}
                    placeholder="Stream name, e.g. Blue"
                    className="flex-1"
                />
                <Button type="button" onClick={addName} variant="outline" size="icon">
                    <Plus className="h-4 w-4" />
                </Button>
            </div>

            <div className="flex flex-wrap gap-2">
                {names.map((n) => (
                    <Badge key={n} variant="secondary" className="gap-1 py-1 pr-1 pl-2">
                        {n}
                        <button
                            onClick={() => removeName(n)}
                            className="hover:text-destructive ml-0.5 inline-flex items-center rounded-full"
                            aria-label={`Remove ${n}`}
                        >
                            <Trash2 className="h-3 w-3" />
                        </button>
                    </Badge>
                ))}
            </div>

            {names.length > 0 && (
                <div className="bg-muted text-muted-foreground rounded-md p-3 text-sm">
                    Preview: <span className="text-foreground font-medium">{preview}</span>
                </div>
            )}

            <div className="pt-2">
                <Button onClick={handleSubmit} disabled={names.length === 0 || isPending}>
                    {isPending ? "Creating..." : "Create Streams"}
                </Button>
            </div>

            {me?.active_school_id && (
                <p className="text-muted-foreground text-xs">
                    Active school: {me.active_school_id}
                </p>
            )}
        </div>
    );
}
