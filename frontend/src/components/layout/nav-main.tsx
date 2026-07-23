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
    GraduationCapIcon,
    ClipboardCheckIcon,
    BarChart3Icon,
    CalendarCheckIcon,
    CalendarDays,
    AlertTriangleIcon,
    HeartPulse,
    DollarSignIcon,
} from "lucide-react";

interface NavItem {
    title: string;
    url: string;
    icon: React.ReactNode;
    isActive?: boolean;
    items?: { title: string; url: string }[];
}

function buildNavItems(role: string): NavItem[] {
    if (!role) return [];

    const items: NavItem[] = [
        {
            title: "Dashboard",
            url: "/",
            icon: <LayoutDashboardIcon className="size-4" />,
            isActive: true,
        },
        {
            title: "Members",
            url: "#",
            icon: <UsersIcon className="size-4" />,
            items: [
                { title: "Admins", url: "/admins" },
                { title: "Teachers", url: "/teachers" },
                { title: "Nurses", url: "/nurses" },
                { title: "Finance", url: "/finance" },
                { title: "Parents", url: "/parents" },
                { title: "Students", url: "/students" },
            ],
        },
        {
            title: "Curriculum",
            url: "/curriculum",
            icon: <BookOpenIcon className="size-4" />,
        },
        {
            title: "Classes",
            url: "/classes",
            icon: <GraduationCapIcon className="size-4" />,
        },
        {
            title: "Time table",
            url: "/timetable",
            icon: <CalendarDays className="size-4" />,
        },
        {
            title: "Attendance",
            url: "#",
            icon: <CalendarCheckIcon className="size-4" />,
            items: [
                { title: "Register", url: "/attendance" },
                { title: "History", url: "/attendance/history" },
            ],
        },
        {
            title: "Assessments",
            url: "/assessments",
            icon: <ClipboardCheckIcon className="size-4" />,
        },
        {
            title: "Reports",
            url: "/reports",
            icon: <BarChart3Icon className="size-4" />,
        },
        {
            title: "Behavior",
            url: "/behavior",
            icon: <AlertTriangleIcon className="size-4" />,
        },
        {
            title: "Finance",
            url: "#",
            icon: <DollarSignIcon className="size-4" />,
            items: [
                { title: "Fee Categories", url: "/finance/fee-categories" },
                { title: "Fee Templates", url: "/finance/fee-templates" },
                { title: "Invoices", url: "/finance/invoices" },
            ],
        },
        {
            title: "Health",
            url: "/health",
            icon: <HeartPulse className="size-4" />,
        },
        {
            title: "Settings",
            url: "#",
            icon: <Settings2Icon className="size-4" />,
            items: [
                { title: "General", url: "/settings" },
                { title: "Academic Years", url: "/academic-years" },
                { title: "Grading Scales", url: "/settings/grading-scales" },
            ],
        },
    ];

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
