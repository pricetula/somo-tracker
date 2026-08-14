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
// import { SchoolSwitcher } from "@/features/school/components/school-switcher";

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
    const { data: me } = useMe();

    const userDisplayName = me ? me.full_name || me.email || "User" : "User";

    return (
        <Sidebar collapsible="icon" {...props}>
            <SidebarHeader>{/*<SchoolSwitcher />*/}</SidebarHeader>
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
