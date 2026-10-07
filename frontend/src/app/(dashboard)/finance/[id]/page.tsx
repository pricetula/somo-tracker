"use client";

import { useParams } from "next/navigation";
import { FinanceDetail } from "@/features/finance";

export default function FinanceDetailPage() {
    const params = useParams();
    const id = params?.id as string;

    return <FinanceDetail id={id} />;
}
