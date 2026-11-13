"use client"

import * as React from "react"

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar"
import { ChevronsUpDownIcon, PlusIcon, SchoolIcon } from "lucide-react"
import type { MeResponse } from "@/features/auth/types"

export function SchoolSwitcher({ me }: { me?: MeResponse | null }) {
  const { isMobile } = useSidebar()
  const schools = me?.schools ?? []
  const activeSchool = me?.active_school

  const [activeSchoolId] = React.useState<string | null>(
    activeSchool?.id ?? null
  )

  if (schools.length === 0) {
    return null
  }

  const handleSwitchSchool = async (schoolId: string) => {
    // Call backend to switch active school
    await fetch("/api/v1/me/active-school", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ school_id: schoolId }),
      credentials: "include",
    })
    // Hard refresh after switching schools
    window.location.reload()
  }

  const currentSchool = schools.find((s) => s.id === (activeSchoolId ?? ""))

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger render={<SidebarMenuButton size="lg" />}>
            <div className="flex aspect-square items-center justify-center">
              <SchoolIcon />
            </div>
            <div className="grid flex-1 text-left leading-tight">
              <span className="truncate font-medium">
                {currentSchool?.name ?? "School"}
              </span>
              <span className="truncate text-xs">Switch school</span>
            </div>
            <ChevronsUpDownIcon className="ml-auto size-4" />
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-fit min-w-[200px]"
            align="start"
            side={isMobile ? "bottom" : "right"}
            sideOffset={4}
          >
            <DropdownMenuGroup>
              <DropdownMenuLabel className="text-muted-foreground">
                Schools
              </DropdownMenuLabel>
              {schools.map((school) => {
                const schoolId = school.id ?? ""
                const isActive =
                  schoolId === (activeSchoolId ?? "") && schoolId !== ""
                return (
                  <DropdownMenuItem
                    key={schoolId}
                    onClick={() => handleSwitchSchool(schoolId)}
                  >
                    {school.name}
                    {isActive && (
                      <span className="ml-auto text-xs">Active</span>
                    )}
                  </DropdownMenuItem>
                )
              })}
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => {}}>
              <PlusIcon className="size-4" />
              <span className="font-medium text-muted-foreground">
                Add school
              </span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
