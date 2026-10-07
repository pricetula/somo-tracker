"use client";

interface GuardianDetailProps {
    id: string;
}

export function GuardianDetail({ id }: GuardianDetailProps) {
    return (
        <div className="p-6">
            <h1 className="text-2xl font-semibold">Guardian</h1>
            <p className="mt-2">Guardian ID: {id}</p>
            <p className="text-muted-foreground mt-1">Name placeholder</p>
        </div>
    );
}
