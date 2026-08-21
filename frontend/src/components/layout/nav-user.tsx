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
import { Skeleton } from "@/components/ui/skeleton";
import { useRouter } from "next/navigation";
import {
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    useSidebar,
} from "@/components/ui/sidebar";
import { useMe } from "@/hooks/use-auth";

export function NavUser() {
    const router = useRouter();
    const { isMobile } = useSidebar();
    const { data: me, isLoading } = useMe();

    const initials = React.useMemo(() => {
        if (!me?.full_name?.length) return "U";
        return me.full_name
            .split(" ")
            .map((name) => {
                const trimmedName = name.trim();
                if (!trimmedName.length) return "";
                return trimmedName[0].toUpperCase();
            })
            .join("");
    }, [me]);

    if (isLoading) {
        return <Skeleton className="h-12" />;
    }

    return (
        <SidebarMenu>
            <SidebarMenuItem>
                <DropdownMenu>
                    <DropdownMenuTrigger
                        render={<SidebarMenuButton size="lg" className="aria-expanded:bg-muted" />}
                    >
                        <Avatar>
                            {/*<AvatarImage src={user.avatar} alt={user.name} />*/}
                            <AvatarFallback>{initials}</AvatarFallback>
                        </Avatar>
                        <div className="grid flex-1 text-left text-sm leading-tight">
                            <span className="truncate font-medium">{me?.full_name || "-"}</span>
                            <span className="truncate text-xs">{me?.email || "-"}</span>
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
                                <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                                    <Avatar>
                                        {/*<AvatarImage src={user.avatar} alt={user.name} />*/}
                                        <AvatarFallback>{initials}</AvatarFallback>
                                    </Avatar>
                                    <div className="grid flex-1 text-left text-sm leading-tight">
                                        <span className="truncate font-medium">
                                            {me?.full_name || "-"}
                                        </span>
                                        <span className="truncate text-xs">{me?.email || "-"}</span>
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
