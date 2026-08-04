"use client";

import { WelcomeGreeting } from "./welcome-greeting";

export function SchoolAdminDashboardPage() {
    return (
        <article>
            <WelcomeGreeting />
            <p className="text-muted-foreground mt-1">School-wide overview and quick access.</p>
        </article>
    );
}
