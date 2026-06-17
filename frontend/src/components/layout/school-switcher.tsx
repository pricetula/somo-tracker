"use client";

import * as React from "react";
import Link from "next/link";

import {
    DropdownMenu,
    DropdownMenuContent,
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
import { ChevronsUpDownIcon, PlusIcon } from "lucide-react";
import type { School } from "@/lib/api/schools";

export function SchoolSwitcher({
    schools,
    activeSchoolId,
    canCreate,
    onSchoolChange,
}: {
    schools: School[];
    activeSchoolId?: string;
    canCreate: boolean;
    onSchoolChange?: (school: School) => void;
}) {
    const { isMobile } = useSidebar();
    const activeSchool = schools.find((s) => s.id === activeSchoolId) ?? schools[0];

    if (!activeSchool) {
        return null;
    }

    const firstName = (activeSchool.name ?? "S").charAt(0).toUpperCase();

    return (
        <SidebarMenu>
            <SidebarMenuItem>
                <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                        <SidebarMenuButton
                            size="lg"
                            className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                        >
                            <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg text-sm font-medium">
                                {firstName}
                            </div>
                            <div className="grid flex-1 text-left text-sm leading-tight">
                                <span className="truncate font-medium">
                                    {activeSchool.name ?? "School"}
                                </span>
                                {activeSchool.is_demo && (
                                    <span className="text-muted-foreground truncate text-xs">
                                        Demo
                                    </span>
                                )}
                            </div>
                            <ChevronsUpDownIcon className="ml-auto" />
                        </SidebarMenuButton>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                        className="w-fit min-w-[200px]"
                        align="start"
                        side={isMobile ? "bottom" : "right"}
                        sideOffset={4}
                    >
                        <DropdownMenuLabel className="text-muted-foreground text-xs">
                            Schools
                        </DropdownMenuLabel>
                        {schools.map((school) => (
                            <DropdownMenuItem
                                key={school.id}
                                onClick={() => onSchoolChange?.(school)}
                                className="gap-2 p-2"
                            >
                                <div className="flex size-6 items-center justify-center rounded-md border text-xs font-medium">
                                    {(school.name ?? "S").charAt(0).toUpperCase()}
                                </div>
                                <span className="flex-1 truncate">{school.name ?? school.id}</span>
                                {school.is_demo && (
                                    <span className="text-muted-foreground text-xs">Demo</span>
                                )}
                            </DropdownMenuItem>
                        ))}
                        {canCreate && (
                            <>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem className="gap-2 p-2" asChild>
                                    <Link href="/schools/new">
                                        <div className="flex size-6 items-center justify-center rounded-md border bg-transparent">
                                            <PlusIcon className="size-4" />
                                        </div>
                                        <span className="text-muted-foreground font-medium">
                                            Add school
                                        </span>
                                    </Link>
                                </DropdownMenuItem>
                            </>
                        )}
                    </DropdownMenuContent>
                </DropdownMenu>
            </SidebarMenuItem>
        </SidebarMenu>
    );
}
