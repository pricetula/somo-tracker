"use client";

import { useParams } from "next/navigation";
import { SubjectDetail } from "@/features/curriculum";

export default function SubjectDetailPage() {
    const params = useParams();
    const gradeId = params.id as string;
    const subjectId = params.subjectId as string;

    return <SubjectDetail gradeId={gradeId} subjectId={subjectId} />;
}
