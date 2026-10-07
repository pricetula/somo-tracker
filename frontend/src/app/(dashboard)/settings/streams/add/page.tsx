"use client";

import { StreamForm } from "@/features/streams";
import { useRouter } from "next/navigation";

export default function AddStreamPage() {
    const router = useRouter();
    return (
        <div className="mx-auto max-w-xl space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Add Stream</h1>
            <StreamForm onSuccess={() => router.push("/settings/streams")} />
        </div>
    );
}
