"use client";

import { useSearchParams } from "next/navigation";

interface AllocationFormProps {
    onSuccess?: () => void;
}

export function AllocationForm({ onSuccess }: AllocationFormProps) {
    const searchParams = useSearchParams();

    const blockId = searchParams.get("block") ?? "";

    const day = searchParams.get("day") ?? "";
    return `allocation ${blockId} = ${day}`;
}
