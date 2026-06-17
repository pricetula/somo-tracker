"use client";

import * as React from "react";

import { NavMain } from "@/components/layout/nav-main";
import { NavUser } from "@/components/layout/nav-user";
import { SchoolSwitcher } from "@/components/layout/school-switcher";
import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarHeader,
    SidebarRail,
} from "@/components/ui/sidebar";
import { useMe } from "@/hooks/use-auth";
import { useSchools, useActivateSchool } from "@/hooks/use-schools";

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
    const { data: me } = useMe();
    const { data: schools = [] } = useSchools(me?.tenant_id);

    const userDisplayName = me
        ? [me.first_name, me.last_name].filter(Boolean).join(" ") || me.email || "User"
        : "User";

    const canCreate = me?.role === "SCHOOL_ADMIN" || me?.role === "SYSTEM_ADMIN";
    const { mutate: switchSchool } = useActivateSchool();

    return (
        <Sidebar collapsible="icon" {...props}>
            <SidebarHeader>
                <SchoolSwitcher
                    schools={schools}
                    activeSchoolId={me?.school_id}
                    canCreate={canCreate}
                    onSchoolChange={(school) => school.id && switchSchool(school.id)}
                />
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
