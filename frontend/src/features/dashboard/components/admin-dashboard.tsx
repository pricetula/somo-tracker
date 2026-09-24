import React from "react";
import { Separator } from "@/components/ui/separator";
import { StudentSummaryCard } from "@/features/students";

export function AdminDashboard() {
    return (
        <article>
            <header className="flex items-center gap-4">
                <StudentSummaryCard />
                <Separator orientation="vertical" />
                <div>ss</div>
            </header>
        </article>
    );
}
