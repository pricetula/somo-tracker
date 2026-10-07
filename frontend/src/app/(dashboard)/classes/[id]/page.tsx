"use client";

import { useParams } from "next/navigation";
import { ClassDetail } from "@/features/classes";

export default function ClassDetailPage() {
    const params = useParams();
    const id = params.id as string;

    return <ClassDetail id={id} />;
}
