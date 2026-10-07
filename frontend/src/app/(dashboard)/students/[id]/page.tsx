"use client";

import { useParams } from "next/navigation";
import { StudentDetail } from "@/features/students";

export default function StudentDetailPage() {
    const params = useParams();
    const id = params?.id as string;

    return <StudentDetail id={id} />;
}
