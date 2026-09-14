"use client";

import { useParams } from "next/navigation";
import { TopicDetail } from "@/features/curriculum";

export default function TopicDetailPage() {
    const params = useParams();
    const subjectId = params.id as string;
    const topicId = params.subjectId as string;
    const subTopicId = params.topicId as string;

    return <TopicDetail subjectId={subjectId} topicId={topicId} subTopicId={subTopicId} />;
}
