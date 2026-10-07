"use client";

import { GuardiansTable } from "@/features/guardians";

export default function GuardiansPage() {
    return (
        <div className="space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Guardians</h1>
            <GuardiansTable />
        </div>
    );
}
