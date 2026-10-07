"use client";

import { useMeSession } from "@/features/auth/hooks/use-me-session";
import { UserCountsPanel } from "./user-counts-panel";

export function AdminDashboard() {
    const { data: me, isLoading } = useMeSession();
    if (isLoading) return <div>Loading...</div>;
    if (!me?.active_school_id) return <div>No active school selected.</div>;
    return (
        <div className="space-y-8">
            <UserCountsPanel schoolId={me.active_school_id} />
        </div>
    );
}
