"use client";

import Link from "next/link";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
    SidebarGroup,
    SidebarGroupLabel,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarMenuSub,
    SidebarMenuSubButton,
    SidebarMenuSubItem,
} from "@/components/ui/sidebar";
import {
    LayoutDashboardIcon,
    UsersIcon,
    Settings2Icon,
    ChevronRightIcon,
    BookOpenIcon,
} from "lucide-react";

interface NavItem {
    title: string;
    url: string;
    icon: React.ReactNode;
    isActive?: boolean;
    items?: { title: string; url: string }[];
}

function buildNavItems(role: string): NavItem[] {
    const isAdmin = role === "SCHOOL_ADMIN" || role === "SYSTEM_ADMIN";
    const canAccessCurriculum = isAdmin || role === "TEACHER";

    const items: NavItem[] = [
        {
            title: "Dashboard",
            url: "/",
            icon: <LayoutDashboardIcon className="size-4" />,
            isActive: true,
        },
    ];

    if (canAccessCurriculum) {
        items.push({
            title: "Curriculum",
            url: "/curriculum",
            icon: <BookOpenIcon className="size-4" />,
        });
    }

    if (isAdmin) {
        items.push(
            {
                title: "Members",
                url: "#",
                icon: <UsersIcon className="size-4" />,
                items: [
                    { title: "Admins", url: "/admins" },
                    { title: "Teachers", url: "/teachers" },
                    { title: "Nurses", url: "/nurses" },
                    { title: "Finance", url: "/finance" },
                    { title: "Students", url: "/students" },
                ],
            },
            {
                title: "Settings",
                url: "#",
                icon: <Settings2Icon className="size-4" />,
                items: [{ title: "General", url: "/settings" }],
            }
        );
    }

    return items;
}

export function NavMain({ role }: { role: string }) {
    const items = buildNavItems(role);

    return (
        <SidebarGroup>
            <SidebarGroupLabel>Platform</SidebarGroupLabel>
            <SidebarMenu>
                {items.map((item) =>
                    item.items ? (
                        <Collapsible
                            key={item.title}
                            asChild
                            defaultOpen={item.isActive}
                            className="group/collapsible"
                        >
                            <SidebarMenuItem>
                                <CollapsibleTrigger asChild>
                                    <SidebarMenuButton tooltip={item.title}>
                                        {item.icon}
                                        <span>{item.title}</span>
                                        <ChevronRightIcon className="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                                    </SidebarMenuButton>
                                </CollapsibleTrigger>
                                <CollapsibleContent>
                                    <SidebarMenuSub>
                                        {item.items.map((subItem) => (
                                            <SidebarMenuSubItem key={subItem.title}>
                                                <SidebarMenuSubButton asChild>
                                                    <Link href={subItem.url}>
                                                        <span>{subItem.title}</span>
                                                    </Link>
                                                </SidebarMenuSubButton>
                                            </SidebarMenuSubItem>
                                        ))}
                                    </SidebarMenuSub>
                                </CollapsibleContent>
                            </SidebarMenuItem>
                        </Collapsible>
                    ) : (
                        <SidebarMenuItem key={item.title}>
                            <SidebarMenuButton asChild tooltip={item.title}>
                                <Link href={item.url}>
                                    {item.icon}
                                    <span>{item.title}</span>
                                </Link>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                    )
                )}
            </SidebarMenu>
        </SidebarGroup>
    );
}
