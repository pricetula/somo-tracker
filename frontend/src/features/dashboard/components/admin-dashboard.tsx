"use client";

import { UserCountsPanel } from "./user-counts-panel";
import { SetupFlow } from "./setup/setup-flow";

export function AdminDashboard() {
    return (
        <>
            <UserCountsPanel />
            <SetupFlow />
        </>
    );
}
