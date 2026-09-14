"use client";

import { useParams } from "next/navigation";
import { TeacherDetail } from "@/features/teachers";

export default function TeacherDetailPage() {
    const params = useParams();
    const id = params?.id as string;

    return <TeacherDetail id={id} />;
}
