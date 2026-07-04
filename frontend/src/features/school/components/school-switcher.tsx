/**
 * School Switcher — dropdown menu to view and switch between schools,
 * and to add a new school.
 *
 * Renders inside the sidebar header. Shows the current active school
 * and a dropdown list of all available schools in the tenant.
 */

"use client";

import * as React from "react";

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
import {
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    useSidebar,
} from "@/components/ui/sidebar";
import { Badge } from "@/components/ui/badge";
import { ChevronsUpDownIcon, Plus } from "lucide-react";

import { useMe } from "@/hooks/use-auth";
import { useSchools, useSetActiveSchool } from "../hooks/use-schools";
import { CreateSchoolDialog } from "./create-school-dialog";

export function SchoolSwitcher() {
    const { isMobile } = useSidebar();
    const { data: me } = useMe();
    const { data: schoolsData } = useSchools();
    const { mutate: switchSchool, isPending: isSwitching } = useSetActiveSchool();
    const [dialogOpen, setDialogOpen] = React.useState(false);

    const userName = me?.full_name ?? me?.email ?? "User";
    const activeSchoolName = me?.school_name ?? "School";
    const activeSchoolId = me?.school_id ?? "";
    const schools = schoolsData?.items ?? [];

    const initials = activeSchoolName
        .split(" ")
        .map((n) => n.charAt(0))
        .join("")
        .toUpperCase()
        .slice(0, 2);

    return (
        <>
            <SidebarMenu>
                <SidebarMenuItem>
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <SidebarMenuButton
                                size="lg"
                                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                            >
                                <Avatar className="h-8 w-8 rounded-lg">
                                    <AvatarFallback className="rounded-lg">
                                        {initials}
                                    </AvatarFallback>
                                </Avatar>
                                <div className="grid flex-1 text-left text-sm leading-tight">
                                    <span className="truncate font-medium">{activeSchoolName}</span>
                                    <span className="truncate text-xs">{userName}</span>
                                </div>
                                <ChevronsUpDownIcon className="ml-auto size-4" />
                            </SidebarMenuButton>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent
                            className="w-fit min-w-56"
                            side={isMobile ? "bottom" : "right"}
                            align="start"
                            sideOffset={4}
                        >
                            <DropdownMenuLabel className="text-muted-foreground text-xs font-medium">
                                Schools
                            </DropdownMenuLabel>
                            <DropdownMenuGroup>
                                {schools.length === 0 ? (
                                    <div className="text-muted-foreground px-2 py-3 text-center text-xs">
                                        No schools yet
                                    </div>
                                ) : (
                                    schools.map((school) => (
                                        <DropdownMenuItem
                                            key={school.id}
                                            disabled={school.id === activeSchoolId || isSwitching}
                                            className="flex items-center gap-2"
                                            onClick={() => switchSchool(school.id)}
                                        >
                                            <Avatar className="h-7 w-7 rounded-md">
                                                <AvatarFallback className="rounded-md text-xs">
                                                    {school.name
                                                        .split(" ")
                                                        .map((n) => n.charAt(0))
                                                        .join("")
                                                        .toUpperCase()
                                                        .slice(0, 2)}
                                                </AvatarFallback>
                                            </Avatar>
                                            <span className="flex-1 truncate text-sm">
                                                {school.name}
                                            </span>
                                            {school.id === activeSchoolId && (
                                                <Badge
                                                    variant="secondary"
                                                    className="bg-sky-100 text-[10px] text-sky-700 dark:bg-sky-900/30 dark:text-sky-400"
                                                >
                                                    Active
                                                </Badge>
                                            )}
                                        </DropdownMenuItem>
                                    ))
                                )}
                            </DropdownMenuGroup>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                                onClick={() => setDialogOpen(true)}
                                className="text-muted-foreground flex items-center gap-2"
                            >
                                <Plus className="size-4" />
                                <span>Add School</span>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </SidebarMenuItem>
            </SidebarMenu>

            <CreateSchoolDialog open={dialogOpen} onOpenChange={setDialogOpen} />
        </>
    );
}
