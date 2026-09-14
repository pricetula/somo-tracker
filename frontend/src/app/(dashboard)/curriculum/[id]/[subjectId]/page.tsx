"use client";

import { useParams } from "next/navigation";
import { SubjectDetail } from "@/features/curriculum";

export default function SubjectDetailPage() {
    const params = useParams();
    const subjectId = params.id as string;
    const topicId = params.subjectId as string;

    return <SubjectDetail subjectId={subjectId} topicId={topicId} />;
}
