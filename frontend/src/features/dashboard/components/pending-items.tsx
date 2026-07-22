/**
 * PendingItems — lists items that need the user's attention.
 *
 * Accepts items as a prop so it can be reused across roles.
 * When nothing needs attention, renders nothing (empty state).
 */

"use client";

import Link from "next/link";
import { AlertTriangleIcon } from "lucide-react";

export interface PendingItemData {
    type: string;
    label: string;
    description: string;
    href: string;
}

interface PendingItemsProps {
    items: PendingItemData[];
    isLoading?: boolean;
}

export function PendingItems({ items, isLoading }: PendingItemsProps) {
    if (isLoading) {
        return (
            <section>
                <h2 className="mb-3 text-lg font-medium">Needs Attention</h2>
                <div className="bg-muted h-16 w-full animate-pulse rounded-lg" />
            </section>
        );
    }

    if (items.length === 0) return null;

    return (
        <section>
            <h2 className="mb-3 text-lg font-medium">Needs Attention</h2>
            <div className="space-y-3">
                {items.map((item) => (
                    <Link
                        key={item.type}
                        href={item.href}
                        className="bg-destructive/10 hover:bg-destructive/15 flex items-start gap-3 rounded-lg px-4 py-3 transition-colors"
                    >
                        <AlertTriangleIcon className="text-destructive mt-0.5 size-4 shrink-0" />
                        <div className="min-w-0">
                            <p className="text-sm font-medium">{item.label}</p>
                            <p className="text-muted-foreground text-sm">{item.description}</p>
                        </div>
                    </Link>
                ))}
            </div>
        </section>
    );
}
