"use client";

import * as React from "react";

import { NavMain } from "@/components/layout/nav-main";
import { NavUser } from "@/components/layout/nav-user";
import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarHeader,
    SidebarRail,
} from "@/components/ui/sidebar";
import { useMe } from "@/hooks/use-auth";

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
    const { data: me } = useMe();

    const userDisplayName = me ? me.full_name || me.email || "User" : "User";

    return (
        <Sidebar collapsible="icon" {...props}>
            <SidebarHeader>
                <div className="flex items-center gap-2 px-2 py-1">
                    <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-6 items-center justify-center rounded-lg text-xs font-medium">
                        S
                    </div>
                    <div className="grid flex-1 text-left text-sm leading-tight">
                        <span className="truncate font-medium">SomoTracker</span>
                    </div>
                </div>
            </SidebarHeader>
            <SidebarContent>
                <NavMain role={me?.role ?? ""} />
            </SidebarContent>
            <SidebarFooter>
                <NavUser
                    user={{
                        name: userDisplayName,
                        email: me?.email ?? "",
                    }}
                />
            </SidebarFooter>
            <SidebarRail />
        </Sidebar>
    );
}
