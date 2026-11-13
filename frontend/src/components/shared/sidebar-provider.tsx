"use client"

import { ReactNode } from "react"

import { SidebarProvider } from "@/components/ui/sidebar"

export function ShellSidebarProvider({
  children,
}: {
  children: ReactNode
}) {
  return <SidebarProvider>{children}</SidebarProvider>
}
