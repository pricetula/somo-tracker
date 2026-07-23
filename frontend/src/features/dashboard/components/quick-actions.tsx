/**
 * QuickActions — a grid of shortcut buttons linking to common tasks.
 *
 * Accepts an actions prop so it can be reused across roles.
 * Each action has an icon, label, and href.
 */

"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";

export interface QuickAction {
    label: string;
    description?: string;
    href: string;
    icon: React.ReactNode;
}

interface QuickActionsProps {
    actions: QuickAction[];
}

export function QuickActions({ actions }: QuickActionsProps) {
    if (actions.length === 0) return null;

    return (
        <section>
            <h2 className="mb-3 text-lg font-medium">Quick Actions</h2>
            <div className="flex flex-wrap gap-2">
                {actions.map((action) => (
                    <Button key={action.href} variant="outline" asChild className="gap-2">
                        <Link href={action.href}>
                            {action.icon}
                            <span>{action.label}</span>
                        </Link>
                    </Button>
                ))}
            </div>
        </section>
    );
}
