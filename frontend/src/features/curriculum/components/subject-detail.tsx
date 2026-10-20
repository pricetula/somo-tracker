"use client";

import Link from "next/link";
import { useTopics, useSubTopics } from "../hooks/use-subjects";

interface SubjectDetailProps {
    gradeId: string;
    subjectId: string;
}

export function SubjectDetail({ gradeId, subjectId }: SubjectDetailProps) {
    const { data: topics = [] } = useTopics(subjectId);
    const selectedTopic = topics[0] ?? null;

    const { data: subTopics = [] } = useSubTopics(selectedTopic?.id ?? "");

    if (!selectedTopic) {
        return (
            <div className="space-y-6 p-6">
                <p className="text-muted-foreground">Loading topics…</p>
            </div>
        );
    }

    return (
        <div className="space-y-6 p-6">
            <div className="space-y-1">
                <h1 className="text-2xl font-semibold">{selectedTopic.name}</h1>
                <p className="text-muted-foreground">
                    {selectedTopic.code}
                    {selectedTopic.description && ` • ${selectedTopic.description}`}
                </p>
            </div>
            <div className="space-y-2">
                <h2 className="text-lg font-medium">Topics</h2>
                <ul className="space-y-1">
                    {topics.map((t) => (
                        <li key={t.id}>
                            <Link
                                href={`/curriculum/${gradeId}/${subjectId}/${t.id}`}
                                className="underline underline-offset-4 hover:no-underline"
                            >
                                {t.name} <span className="text-muted-foreground">({t.code})</span>
                            </Link>
                        </li>
                    ))}
                    {topics.length === 0 && (
                        <p className="text-muted-foreground">No topics found.</p>
                    )}
                </ul>
            </div>
            <div className="space-y-2">
                <h2 className="text-lg font-medium">Sub-topics</h2>
                <ul className="space-y-1">
                    {subTopics.map((st) => (
                        <li key={st.id}>
                            <Link
                                href={`/curriculum/${gradeId}/${subjectId}/${selectedTopic.id}/${st.id}`}
                                className="underline underline-offset-4 hover:no-underline"
                            >
                                {st.name} <span className="text-muted-foreground">({st.code})</span>
                            </Link>
                        </li>
                    ))}
                    {subTopics.length === 0 && (
                        <p className="text-muted-foreground">No sub-topics found.</p>
                    )}
                </ul>
            </div>
        </div>
    );
}
