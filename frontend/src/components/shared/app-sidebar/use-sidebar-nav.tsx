/**
 * Pure role-based link filtering for the app sidebar.
 * Testable in isolation — no React rendering required.
 */

import type { MeResponse } from "@/features/auth/types";
import { LayoutDashboard } from "lucide-react";
import { User } from "lucide-react";
import { LayoutGrid } from "lucide-react";
import { BookOpen } from "lucide-react";
import { Calendar } from "lucide-react";
import { CheckCircle } from "lucide-react";
import { CreditCard } from "lucide-react";
import { UserCircle } from "lucide-react";
import { Users } from "lucide-react";
import { BarChart3 } from "lucide-react";
import { LogOut } from "lucide-react";

export interface SidebarLink {
  id: string
  title: string
  url: string
  icon: React.ReactNode
  roles: string[]
}

// All MVP links. Pages not yet implemented are rendered "Coming soon" by NavMain.
const allLinks: SidebarLink[] = [
  {
    id: "dashboard",
    title: "Dashboard",
    url: "/",
    icon: <LayoutDashboard />,
    roles: ["HEADTEACHER", "TEACHER", "GUARDIAN", "ACCOUNTANT"],
  },
  {
    id: "students",
    title: "Students",
    url: "/students",
    icon: <User />,
    roles: ["HEADTEACHER", "TEACHER", "GUARDIAN", "ACCOUNTANT"],
  },
  {
    id: "classes",
    title: "Classes",
    url: "/classes",
    icon: <LayoutGrid />,
    roles: ["HEADTEACHER", "TEACHER"],
  },
  {
    id: "learning-areas",
    title: "Learning Areas",
    url: "/learning-areas",
    icon: <BookOpen />,
    roles: ["HEADTEACHER", "TEACHER"],
  },
  {
    id: "timetable",
    title: "Timetable",
    url: "/timetable",
    icon: <Calendar />,
    roles: ["HEADTEACHER", "TEACHER"],
  },
  {
    id: "attendance",
    title: "Attendance",
    url: "/attendance",
    icon: <CheckCircle />,
    roles: ["HEADTEACHER", "TEACHER"],
  },
  {
    id: "fees",
    title: "Fees / Payments",
    url: "/fees",
    icon: <CreditCard />,
    roles: ["HEADTEACHER", "ACCOUNTANT"],
  },
  {
    id: "guardians",
    title: "Guardians",
    url: "/guardians",
    icon: <UserCircle />,
    roles: ["HEADTEACHER", "TEACHER"],
  },
  {
    id: "staff",
    title: "Staff / Teachers",
    url: "/staff",
    icon: <Users />,
    roles: ["HEADTEACHER"],
  },
  {
    id: "reports",
    title: "Reports",
    url: "/reports",
    icon: <BarChart3 />,
    roles: ["HEADTEACHER"],
  },
  {
    id: "logout",
    title: "Logout",
    url: "/auth/logout",
    icon: <LogOut />,
    roles: ["HEADTEACHER", "TEACHER", "GUARDIAN", "ACCOUNTANT"],
  },
  {
    id: "profile",
    title: "Profile / Account",
    url: "/profile",
    icon: <User />,
    roles: ["HEADTEACHER", "TEACHER", "GUARDIAN", "ACCOUNTANT"],
  },
];

/**
 * Returns the subset of links visible to the given role.
 * Pure function — easy to unit test per role.
 */
export function getSidebarLinksByRole(role: string): SidebarLink[] {
  return allLinks.filter((link) => link.roles.includes(role));
}

/**
 * Hook that returns visible sidebar links for the current user.
 * Falls back to no links when no session data is available.
 */
export function useSidebarNav(me?: MeResponse | null) {
  const role = me?.role ?? "";
  return getSidebarLinksByRole(role);
}
