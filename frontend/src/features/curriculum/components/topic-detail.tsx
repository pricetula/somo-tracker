"use client";

import Link from "next/link";
import { useSubTopics, useSubTopic } from "../hooks/use-subjects";

interface TopicDetailProps {
    _subjectId?: string;
    subjectId?: string;
    subTopicId?: string;
}

export function TopicDetail({ _subjectId, subjectId, subTopicId }: TopicDetailProps) {
    const { data: selectedSubTopic } = useSubTopic(subTopicId ?? "");
    const topicId = selectedSubTopic?.topicId ?? null;

    const { data: subTopics = [] } = useSubTopics(topicId ?? "");

    if (!selectedSubTopic || !topicId) {
        return (
            <div className="space-y-6 p-6">
                <p className="text-muted-foreground">Loading…</p>
            </div>
        );
    }

    return (
        <div className="space-y-6 p-6">
            <div className="space-y-1">
                <h1 className="text-2xl font-semibold">{selectedSubTopic.name}</h1>
                <p className="text-muted-foreground">{selectedSubTopic.code}</p>
            </div>
            <div className="space-y-2">
                <h2 className="text-lg font-medium">Sub-topics</h2>
                <ul className="space-y-1">
                    {subTopics.map((st) => (
                        <li key={st.id}>
                            <Link
                                href={`/curriculum/${_subjectId}/${subjectId}/${st.id}`}
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
            <div className="space-y-2">
                <h2 className="text-lg font-medium">Performance</h2>
                <p className="text-muted-foreground">
                    Performance indicator placeholder for sub-topic.
                </p>
                <div className="space-y-1">
                    <p className="text-muted-foreground">Completion: 72%</p>
                    <p className="text-muted-foreground">Average score: 84/100</p>
                </div>
            </div>
        </div>
    );
}
