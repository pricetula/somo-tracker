"use client";

import * as React from "react";
import { ChevronsUpDownIcon, Plus } from "lucide-react";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    useSidebar,
} from "@/components/ui/sidebar";
import { useMeSession } from "@/features/auth/hooks/use-me-session";
import { useSchoolsList } from "@/features/school/hooks/use-schools-list";
import Link from "next/link";

export function SchoolSwitcher() {
    const { isMobile } = useSidebar();
    const { data: me } = useMeSession();
    const { data: schools = [], isLoading } = useSchoolsList();

    const activeSchool = !me?.active_school_id
        ? null
        : (schools.find((s) => s.id === me.active_school_id) ?? null);

    return (
        <SidebarMenu>
            <SidebarMenuItem>
                <DropdownMenu>
                    <DropdownMenuTrigger
                        render={
                            <SidebarMenuButton
                                size="lg"
                                className="data-[state=open]:bg-muted data-[state=open]:text-foreground"
                            >
                                <div className="grid flex-1 text-left leading-tight">
                                    <span className="truncate">
                                        {isLoading
                                            ? "Loading..."
                                            : (activeSchool?.name ?? "No school")}
                                    </span>
                                </div>
                                <ChevronsUpDownIcon className="ml-auto size-4" />
                            </SidebarMenuButton>
                        }
                    />
                    <DropdownMenuContent
                        align="start"
                        side={isMobile ? "bottom" : "right"}
                        sideOffset={4}
                    >
                        {schools.map((school) => (
                            <DropdownMenuItem key={school.id} className="">
                                {school.name}
                            </DropdownMenuItem>
                        ))}
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                            className="text-muted-foreground"
                            render={
                                <Link href="/schools/new">
                                    <div className="flex size-6 items-center justify-center rounded-md border bg-transparent">
                                        <Plus className="size-4" />
                                    </div>
                                    <div className="text-muted-foreground font-medium">
                                        Add school
                                    </div>
                                </Link>
                            }
                        />
                    </DropdownMenuContent>
                </DropdownMenu>
            </SidebarMenuItem>
        </SidebarMenu>
    );
}
