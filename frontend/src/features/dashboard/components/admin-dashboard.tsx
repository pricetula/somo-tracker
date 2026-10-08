"use client";

import { UserCountsPanel } from "./user-counts-panel";

export function AdminDashboard() {
    return (
        <div className="space-y-8">
            <UserCountsPanel />
        </div>
    );
}
