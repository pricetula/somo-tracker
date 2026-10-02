"use client";

import { useParams } from "next/navigation";
import { TopicDetail } from "@/features/curriculum";

export default function TopicDetailPage() {
    const params = useParams();
    const gradeId = params.id as string;
    const subjectId = params.subjectId as string;
    const subTopicId = params.topicId as string;

    return <TopicDetail _subjectId={gradeId} subjectId={subjectId} subTopicId={subTopicId} />;
}
