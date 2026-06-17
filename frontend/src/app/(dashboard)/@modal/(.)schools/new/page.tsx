"use client";

import { useRouter } from "next/navigation";
import { AddSchoolForm } from "@/features/school";

export default function AddSchoolModal() {
    const router = useRouter();

    return <AddSchoolForm open onOpenChange={() => router.back()} />;
}
