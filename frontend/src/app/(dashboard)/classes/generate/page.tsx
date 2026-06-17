"use client";

import { useRouter } from "next/navigation";
import { ClassStreamGenerator } from "@/features/classes";

export default function ClassesGeneratePage() {
    const router = useRouter();

    return (
        <div className="min-h-screen p-6">
            {/* Centered form */}
            <div className="mx-auto max-w-3xl">
                <ClassStreamGenerator onSuccess={() => router.push("/dashboard")} />
            </div>
        </div>
    );
}
