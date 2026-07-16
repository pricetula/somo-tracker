/**
 * ParentBehaviorView — shows approved behavior notes for the parent's linked children.
 *
 * Fetches linked children from the parent profile and displays any approved
 * behavior notes. Links to each child's full detail page for more information.
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { AlertTriangle, User, ArrowUpRight } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useMe } from "@/hooks/use-auth";
import { getMyParentProfile } from "@/lib/api/parents";

export function ParentBehaviorView() {
    const { data: me } = useMe();

    const { data: parentProfile, isLoading } = useQuery({
        queryKey: ["parent", "me"],
        queryFn: () => getMyParentProfile(),
        enabled: !!me,
    });

    const children = parentProfile?.data?.linked_students ?? [];

    if (isLoading) {
        return (
            <div className="space-y-4">
                {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className="h-20 w-full rounded-lg" />
                ))}
            </div>
        );
    }

    if (children.length === 0) {
        return (
            <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                <User className="h-8 w-8" />
                <p className="font-medium">No linked children</p>
                <p className="max-w-sm text-center">
                    You don&apos;t have any children linked to your account. Contact the school to
                    link your children so you can view behavior notes.
                </p>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <p className="text-muted-foreground">
                Behavior notes logged by teachers for your children. Urgent notes require your
                attention.
            </p>

            {children.map((child) => (
                <div key={child.student_id} className="rounded-lg border p-4">
                    <div className="flex items-start justify-between">
                        <div className="space-y-1">
                            <div className="flex items-center gap-2">
                                <h3 className="font-medium">{child.full_name}</h3>
                                {child.is_primary && (
                                    <Badge variant="secondary" className="text-[10px]">
                                        Primary
                                    </Badge>
                                )}
                            </div>
                            {child.relationship && (
                                <p className="text-muted-foreground text-xs">
                                    {child.relationship}
                                </p>
                            )}
                        </div>
                        <Button variant="outline" size="sm" asChild>
                            <Link href={`/students/${child.student_id}`}>
                                <ArrowUpRight className="mr-1 h-3 w-3" />
                                View Profile
                            </Link>
                        </Button>
                    </div>

                    <div className="mt-3">
                        <p className="text-muted-foreground text-xs">
                            Behavior notes appear in your child&apos;s student profile and term
                            report. Urgent notes are flagged and communicated directly by the
                            school.
                        </p>
                    </div>

                    <div className="mt-3">
                        <Button variant="ghost" size="sm" asChild>
                            <Link href={`/students/${child.student_id}`}>
                                <AlertTriangle className="mr-1 h-3 w-3" />
                                View behavior history
                            </Link>
                        </Button>
                    </div>
                </div>
            ))}
        </div>
    );
}
