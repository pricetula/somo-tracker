"use client";

import { useMe } from "@/hooks/use-auth";

export function TeacherDashboardPage() {
    const { data: me } = useMe();

    return (
        <div className="flex flex-1 flex-col gap-6 p-6">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                    Welcome back{me?.first_name ? `, ${me.first_name}` : ""}
                </h1>
                <p className="text-muted-foreground mt-1 text-sm">Welcome to SomoTracker.</p>
            </div>
        </div>
    );
}
