"use client";

import Link from "next/link";
import { useSubject, useTopics } from "../hooks/use-subjects";

interface CurriculumDetailProps {
    id: string;
}

export function CurriculumDetail({ id }: CurriculumDetailProps) {
    const { data: subject, isLoading: subjectLoading } = useSubject(id);
    const { data: topics = [] } = useTopics(id);

    if (subjectLoading || !subject) return null;

    return (
        <div className="space-y-6 p-6">
            <div className="space-y-1">
                <h1 className="text-2xl font-semibold">{subject.name}</h1>
                <p className="text-muted-foreground">
                    {subject.code} • {subject.grade}
                </p>
            </div>
            <div className="space-y-2">
                <h2 className="text-lg font-medium">Topics</h2>
                <ul className="space-y-1">
                    {topics.map((t) => (
                        <li key={t.id}>
                            <Link
                                href={`/curriculum/${id}/${t.id}`}
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
        </div>
    );
}
