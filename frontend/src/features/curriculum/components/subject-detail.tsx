"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { listTopics, listSubTopics } from "../services/api";
import type { TopicItem, SubTopicItem } from "../types/curriculum";

interface SubjectDetailProps {
    gradeId: string;
    subjectId: string;
}

export function SubjectDetail({ gradeId, subjectId }: SubjectDetailProps) {
    const [topics, setTopics] = useState<TopicItem[]>([]);
    const [selectedTopic, setSelectedTopic] = useState<TopicItem | null>(null);
    const [subTopics, setSubTopics] = useState<SubTopicItem[]>([]);

    useEffect(() => {
        listTopics(subjectId).then((data) => {
            setTopics(data);
            if (data.length > 0 && !selectedTopic) {
                setSelectedTopic(data[0]);
            }
        });
    }, [subjectId, selectedTopic]);

    useEffect(() => {
        if (selectedTopic) {
            listSubTopics(selectedTopic.id).then(setSubTopics);
        }
    }, [selectedTopic]);

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
