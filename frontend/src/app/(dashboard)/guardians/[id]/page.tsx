"use client";

import { useParams } from "next/navigation";
import { GuardianDetail } from "@/features/guardians";

export default function GuardianDetailPage() {
    const params = useParams();
    const id = params?.id as string;

    return <GuardianDetail id={id} />;
}
