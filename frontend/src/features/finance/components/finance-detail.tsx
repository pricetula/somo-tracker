"use client";

interface FinanceDetailProps {
    id: string;
}

export function FinanceDetail({ id }: FinanceDetailProps) {
    return (
        <div className="p-6">
            <h1 className="text-2xl font-semibold">Finance</h1>
            <p className="mt-2">Finance ID: {id}</p>
            <p className="text-muted-foreground mt-1">Name placeholder</p>
        </div>
    );
}
