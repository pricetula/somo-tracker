"use client";

import { useEffect, useState } from "react";
import { getSubTopic } from "../services/api";
import type { SubTopicItem } from "../types/curriculum";

interface TopicDetailProps {
    subjectId: string;
    topicId: string;
    subTopicId: string;
}

export function TopicDetail({ _subjectId, topicId, subTopicId }: TopicDetailProps) {
    const [subTopic, setSubTopic] = useState<SubTopicItem | null>(null);

    useEffect(() => {
        getSubTopic(topicId, subTopicId).then(setSubTopic);
    }, [topicId, subTopicId]);

    if (!subTopic) return null;

    return (
        <div className="space-y-6 p-6">
            <div className="space-y-1">
                <h1 className="text-2xl font-semibold">{subTopic.name}</h1>
                <p className="text-muted-foreground">{subTopic.code}</p>
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
