"use client";

import { use } from "react";
import { StreamForm } from "@/features/streams";
import { useRouter } from "next/navigation";

export default function StreamDetailPage({ params }: { params: Promise<{ id: string }> }) {
    const { id } = use(params);
    const router = useRouter();
    return (
        <div className="mx-auto max-w-xl space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Edit Stream</h1>
            <StreamForm streamId={id} onSuccess={() => router.push("/settings/streams")} />
        </div>
    );
}
