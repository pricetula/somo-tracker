"use client";

import * as React from "react";

import { NavMain } from "@/components/shared/app-sidebar/nav-main";
import { NavUser } from "@/components/shared/app-sidebar/nav-user";
import { SchoolSwitcher } from "@/components/shared/app-sidebar/school-switcher";
import type { MeResponse } from "@/features/auth/types";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar";

export function AppSidebar({
  me,
  ...props
}: React.ComponentProps<typeof Sidebar> & { me?: MeResponse | null }) {
  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <SchoolSwitcher me={me} />
      </SidebarHeader>
      <SidebarContent>
        <NavMain me={me} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={me} />
      </SidebarFooter>
      <SidebarRail className="after:bg-zinc-50 hover:after:bg-zinc-100 transition-colors" />
    </Sidebar>
  );
}
