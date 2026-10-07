"use client";

import React from "react";
import { ChevronsUpDownIcon, CreditCardIcon, BellIcon, LogOutIcon } from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useRouter } from "next/navigation";
import {
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    useSidebar,
} from "@/components/ui/sidebar";
import { Skeleton } from "@/components/ui/skeleton";
import { useMeSession } from "@/features/auth/hooks/use-me-session";

function getInitials(name: string): string {
    const parts = name.trim().split(/\s+/);
    if (parts.length === 0) return "U";
    if (parts.length === 1) return parts[0].charAt(0).toUpperCase();
    return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase();
}

export function NavUser() {
    const router = useRouter();
    const { isMobile } = useSidebar();
    const { data: me, isLoading } = useMeSession();
    const initials = me ? getInitials(me.user_name) : "U";

    if (isLoading) {
        return (
            <SidebarMenu>
                <SidebarMenuItem>
                    <div className="flex items-center gap-2 px-2 py-2">
                        <Skeleton className="h-9 w-9 rounded-full" />
                        <div className="grid flex-1 gap-1.5">
                            <Skeleton className="h-4 w-24" />
                            <Skeleton className="h-3 w-32" />
                        </div>
                    </div>
                </SidebarMenuItem>
            </SidebarMenu>
        );
    }

    return (
        <SidebarMenu>
            <SidebarMenuItem>
                <DropdownMenu>
                    <DropdownMenuTrigger
                        render={<SidebarMenuButton size="lg" className="aria-expanded:bg-muted" />}
                    >
                        <Avatar>
                            <AvatarFallback>{initials}</AvatarFallback>
                        </Avatar>
                        <div className="grid flex-1 text-left leading-tight">
                            <span className="truncate font-medium">{me?.user_name ?? "User"}</span>
                            <span className="truncate text-xs">{me?.email ?? "Signed in"}</span>
                        </div>
                        <ChevronsUpDownIcon className="ml-auto size-4" />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                        className="w-fit"
                        side={isMobile ? "bottom" : "right"}
                        align="end"
                        sideOffset={4}
                    >
                        <DropdownMenuGroup>
                            <DropdownMenuLabel className="p-0 font-normal">
                                <div className="flex items-center gap-2 px-1 py-1.5 text-left">
                                    <Avatar>
                                        <AvatarFallback>{initials}</AvatarFallback>
                                    </Avatar>
                                    <div className="grid flex-1 text-left leading-tight">
                                        <span className="truncate font-medium">
                                            {me?.user_name ?? "User"}
                                        </span>
                                        <span className="truncate text-xs">
                                            {me?.email ?? "Authenticated"}
                                        </span>
                                    </div>
                                </div>
                            </DropdownMenuLabel>
                        </DropdownMenuGroup>
                        <DropdownMenuSeparator />
                        <DropdownMenuGroup>
                            <DropdownMenuItem>
                                <CreditCardIcon />
                                Billing
                            </DropdownMenuItem>
                            <DropdownMenuItem>
                                <BellIcon />
                                Notifications
                            </DropdownMenuItem>
                        </DropdownMenuGroup>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem onClick={() => router.push("/logout")}>
                            <LogOutIcon />
                            Log out
                        </DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
            </SidebarMenuItem>
        </SidebarMenu>
    );
}
