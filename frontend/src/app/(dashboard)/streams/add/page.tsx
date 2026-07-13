/**
 * Add Stream Page — Full page render for /streams/add.
 *
 * On hard refresh, renders the add stream form as a standalone page.
 * When client-navigated from the streams page, it is intercepted
 * by @modal/(.)streams/add and rendered as a dialog overlay.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { AddStreamForm } from "@/features/streams";

// ─── Props ────────────────────────────────────────────────────────────────

interface AddStreamPageProps {
    searchParams?: Promise<{ value?: string }>;
}

// ─── Page ─────────────────────────────────────────────────────────────────

export default function AddStreamPage(props: AddStreamPageProps) {
    const router = useRouter();
    const searchParams = props.searchParams ? use(props.searchParams) : {};
    const defaultValue = searchParams.value ?? "";

    return (
        <div className="mx-auto max-w-lg p-6">
            <Button variant="ghost" size="sm" onClick={() => router.back()} className="mb-4 -ml-2">
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back
            </Button>
            <h1 className="mb-1 text-lg font-semibold">Add Stream</h1>
            <p className="text-muted-foreground mb-6 text-sm">
                Add a new stream (section) for your school.
            </p>
            <AddStreamForm defaultValue={defaultValue} />
        </div>
    );
}
