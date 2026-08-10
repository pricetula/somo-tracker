"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useMe } from "@/hooks/use-auth";
import { MappedUserRoles } from "@/lib/api/members";

export function WelcomeGreeting() {
    const { data, isLoading } = useMe();
    const name = data?.full_name ? data.full_name : "";
    const role = data?.role ? MappedUserRoles.get(data.role) : "";
    return (
        <Card className="w-full max-w-sm">
            <CardHeader>
                <CardTitle className="text-lg">
                    {isLoading ? (
                        <Skeleton
                            className="inline-block size-4 w-38 rounded-sm"
                            data-sidebar="menu-skeleton-icon"
                        />
                    ) : (
                        <>
                            <span className="text-foreground/30">Welcome, </span>
                            <span>{name}</span>
                        </>
                    )}
                </CardTitle>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton
                        className="inline-block size-4 w-18 rounded-sm"
                        data-sidebar="menu-skeleton-icon"
                    />
                ) : (
                    <span>{role}</span>
                )}
            </CardContent>
        </Card>
    );
}
