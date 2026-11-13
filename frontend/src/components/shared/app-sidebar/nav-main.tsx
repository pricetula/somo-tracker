"use client"

import { useSidebarNav } from "./use-sidebar-nav"
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

export function NavMain({ me }: { me?: { role?: string } | null }) {
  const links = useSidebarNav(me)

  // Filter out user-menu items (logout, profile) — those live in NavUser
  const navLinks = links.filter(
    (link) => link.id !== "logout" && link.id !== "profile"
  )

  return (
    <SidebarGroup>
      <SidebarGroupLabel>Platform</SidebarGroupLabel>
      <SidebarMenu>
        {navLinks.map((link) => (
          <SidebarMenuItem key={link.id}>
            <SidebarMenuButton disabled className="opacity-60 pointer-events-none">
              {link.icon}
              <span>{link.title}</span>
              <span className="ml-auto text-xs opacity-60">Coming soon</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        ))}
      </SidebarMenu>
    </SidebarGroup>
  )
}
