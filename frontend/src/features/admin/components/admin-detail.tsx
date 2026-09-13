"use client";

interface AdminDetailProps {
    id: string;
}

export function AdminDetail({ id }: AdminDetailProps) {
    return (
        <div className="p-6">
            <h1 className="text-2xl font-semibold">Admin</h1>
            <p className="mt-2">Admin ID: {id}</p>
            <p className="text-muted-foreground mt-1">Name placeholder</p>
        </div>
    );
}
