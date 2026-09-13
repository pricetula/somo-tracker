"use client";

import { AdminsTable } from "@/features/admin";

export default function AdminsPage() {
    return (
        <div className="space-y-4 p-6">
            <h1 className="text-2xl font-semibold">Admins</h1>
            <AdminsTable />
        </div>
    );
}
