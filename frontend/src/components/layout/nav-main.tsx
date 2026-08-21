"use client";

import React from "react";
import { useRouter } from "next/navigation";
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
    CalendarDays,
    CalendarCheck,
    AlertTriangleIcon,
    HeartPulse,
    DollarSignIcon,
} from "lucide-react";
import { useMe } from "@/hooks/use-auth";

interface NavItem {
    title: string;
    url: string;
    icon: React.ReactNode;
    isActive?: boolean;
    items?: { title: string; url: string }[];
}

function buildNavItems(role: string): NavItem[] {
    if (!role) return [];

    // ── Parent role: simplified nav ────────────────────────────────
    if (role === "PARENT") {
        return [
            {
                title: "Dashboard",
                url: "/",
                icon: <LayoutDashboardIcon className="size-4" />,
                isActive: true,
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
        ];
    }

    // ── School staff roles ─────────────────────────────────────────
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
            url: "/attendance",
            icon: <CalendarCheck className="size-4" />,
            items: [
                { title: "Sessions", url: "/attendance" },
                { title: "Summaries", url: "/attendance/summaries" },
            ],
        },
        {
            title: "Assessments",
            url: "#",
            icon: <ClipboardCheckIcon className="size-4" />,
            items: [
                { title: "Sessions", url: "/assessments" },
                { title: "Grading Scales", url: "/assessments/grading-scales" },
                { title: "Weight Configs", url: "/assessments/weight-configs" },
            ],
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
            ],
        },
    ];

    return items;
}

export function NavMain() {
    const router = useRouter();
    const { data: me, isLoading } = useMe();
    const items = React.useMemo(() => (me?.role ? buildNavItems(me.role || "") : []), [me]);

    if (isLoading) {
        return (
            <ul>
                <li>loading</li>
            </ul>
        );
    }

    return (
        <SidebarGroup>
            <SidebarGroupLabel>Platform</SidebarGroupLabel>
            <SidebarMenu>
                {items.map((item) =>
                    item?.items ? (
                        <Collapsible
                            key={item.title}
                            defaultOpen={item.isActive}
                            className="group/collapsible"
                            render={<SidebarMenuItem />}
                        >
                            <CollapsibleTrigger render={<SidebarMenuButton tooltip={item.title} />}>
                                {item.icon}
                                <span>{item.title}</span>
                                <ChevronRightIcon className="ml-auto transition-transform duration-200 group-data-open/collapsible:rotate-90" />
                            </CollapsibleTrigger>
                            <CollapsibleContent>
                                <SidebarMenuSub>
                                    {item.items?.map((subItem) => (
                                        <SidebarMenuSubItem key={subItem.title}>
                                            <SidebarMenuSubButton render={<a href={subItem.url} />}>
                                                <span>{subItem.title}</span>
                                            </SidebarMenuSubButton>
                                        </SidebarMenuSubItem>
                                    ))}
                                </SidebarMenuSub>
                            </CollapsibleContent>
                        </Collapsible>
                    ) : (
                        <SidebarMenuItem key={item.title}>
                            <SidebarMenuButton
                                onClick={() => router.push(item.url)}
                                tooltip={item.title}
                            >
                                {item.icon}
                                <span>{item.title}</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                    )
                )}
            </SidebarMenu>
        </SidebarGroup>
    );
}
