"use client";

import { useParams } from "next/navigation";
import { CurriculumDetail } from "@/features/curriculum";

export default function CurriculumDetailPage() {
    const params = useParams();
    const id = params.id as string;

    return <CurriculumDetail id={id} />;
}
