"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { listTopics, listSubTopics } from "../services/api";
import type { TopicItem, SubTopicItem } from "../types/curriculum";

interface SubjectDetailProps {
    subjectId: string;
    topicId: string;
}

export function SubjectDetail({ subjectId, topicId }: SubjectDetailProps) {
    const [topic, setTopic] = useState<TopicItem | null>(null);
    const [subTopics, setSubTopics] = useState<SubTopicItem[]>([]);

    useEffect(() => {
        listTopics(subjectId).then((list) => {
            setTopic(list.find((t) => t.id === topicId) ?? null);
        });
        listSubTopics(topicId).then(setSubTopics);
    }, [subjectId, topicId]);

    if (!topic) return null;

    return (
        <div className="space-y-6 p-6">
            <div className="space-y-1">
                <h1 className="text-2xl font-semibold">{topic.name}</h1>
                <p className="text-muted-foreground">
                    {topic.code}
                    {topic.description && ` • ${topic.description}`}
                </p>
            </div>
            <div className="space-y-2">
                <h2 className="text-lg font-medium">Sub-topics</h2>
                <ul className="space-y-1">
                    {subTopics.map((st) => (
                        <li key={st.id}>
                            <Link
                                href={`/curriculum/${subjectId}/${topicId}/${st.id}`}
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
