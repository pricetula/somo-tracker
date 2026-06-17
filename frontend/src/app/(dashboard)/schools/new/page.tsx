"use client";

import { useRouter } from "next/navigation";
import { AddSchoolForm } from "@/features/school";

export default function AddSchoolPage() {
    const router = useRouter();

    return (
        <div className="flex min-h-[60vh] items-center justify-center">
            <AddSchoolForm open onOpenChange={(open) => !open && router.push("/")} />
        </div>
    );
}
