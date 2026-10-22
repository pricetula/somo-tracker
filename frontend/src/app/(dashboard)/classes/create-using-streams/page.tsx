"use client";

import { CreateUsingStreamsForm } from "@/features/classes/components/create-using-streams-form";

export default function CreateUsingStreamsPage() {
    return (
        <div className="mx-auto max-w-6xl space-y-6 p-6">
            <div>
                <h1 className="text-2xl font-semibold">Auto populate Classrooms</h1>
                <p className="text-muted-foreground text-sm">
                    Derive classrooms from active streams and grade levels. Edit names before
                    creating.
                </p>
            </div>
            <CreateUsingStreamsForm />
        </div>
    );
}
