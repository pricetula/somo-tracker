"use client";

import { useParams } from "next/navigation";
import { AdminDetail } from "@/features/admin";

export default function AdminDetailPage() {
    const params = useParams();
    const id = params?.id as string;

    return <AdminDetail id={id} />;
}
