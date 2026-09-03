"use client";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
    DropdownMenuGroup,
} from "@/components/ui/dropdown-menu";
import {
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    useSidebar,
} from "@/components/ui/sidebar";
import { ChevronsUpDown, Plus, Check } from "lucide-react";
import { useMe } from "@/hooks/use-auth";
import { useSchools, useSetActiveSchool } from "../hooks/use-schools";
import { CreateSchoolDialog } from "./create-school-dialog";
import * as React from "react";

export function SchoolSwitcher() {
    const { isMobile } = useSidebar();
    const { data: me } = useMe();
    const { data: schoolsData } = useSchools();
    const { mutate: switchSchool } = useSetActiveSchool();
    const [dialogOpen, setDialogOpen] = React.useState(false);

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
                        <DropdownMenuTrigger
                            render={
                                <SidebarMenuButton
                                    size="lg"
                                    className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                                >
                                    <Avatar>
                                        {/*<AvatarImage src={user.avatar} alt={user.name} />*/}
                                        <AvatarFallback>{initials}</AvatarFallback>
                                    </Avatar>
                                    <div className="grid flex-1 text-left leading-tight">
                                        <span className="truncate font-medium">
                                            {activeSchoolName}
                                        </span>
                                    </div>
                                    <ChevronsUpDown className="ml-auto" />
                                </SidebarMenuButton>
                            }
                        />
                        <DropdownMenuContent
                            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
                            align="start"
                            side={isMobile ? "bottom" : "right"}
                            sideOffset={4}
                        >
                            <DropdownMenuGroup>
                                <DropdownMenuLabel className="text-muted-foreground">
                                    Teams
                                </DropdownMenuLabel>
                                {schools.map((school) => (
                                    <DropdownMenuItem
                                        key={school.name}
                                        onClick={() => switchSchool(school.id)}
                                        className="gap-2 p-2"
                                    >
                                        <Avatar>
                                            <AvatarFallback>{school.name[0]}</AvatarFallback>
                                        </Avatar>
                                        {school.name}
                                        {activeSchoolId === school.id && <Check />}
                                    </DropdownMenuItem>
                                ))}
                            </DropdownMenuGroup>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                                className="gap-2 p-2"
                                onClick={() => setDialogOpen(true)}
                            >
                                <div className="flex size-6 items-center justify-center rounded-md border bg-transparent">
                                    <Plus className="size-4" />
                                </div>
                                <div className="text-muted-foreground font-medium">Add team</div>
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </SidebarMenuItem>
            </SidebarMenu>

            <CreateSchoolDialog open={dialogOpen} onOpenChange={setDialogOpen} />
        </>
    );
}
