/**
 * InvitationCountBadge — reusable badge showing the number of pending
 * invitations for a given role. Links to the invitations page.
 *
 * Replaces the duplicated inline components previously scattered across
 * nurses, parents, teachers, finance, and admins page files.
 */

"use client";

import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

import { useInvitationCount } from "../hooks/use-invitations";

interface InvitationCountBadgeProps {
    role: string;
    href: string;
}

export function InvitationCountBadge({ role, href }: InvitationCountBadgeProps) {
    const { data, isLoading } = useInvitationCount(role);

    if (isLoading) {
        return <Skeleton className="h-9 w-28" />;
    }

    const count = data?.total ?? 0;
    const label = `${count} ${count === 1 ? "invitation" : "invitations"}`;

    return (
        <Button variant="outline" size="sm" asChild>
            <Link href={href}>{label}</Link>
        </Button>
    );
}
