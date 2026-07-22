/**
 * Role-specific quick action configurations.
 *
 * Each role gets its own set of shortcut buttons on the dashboard.
 * Add or remove actions here — the dashboard components stay generic.
 */

import type { QuickAction } from "./components/quick-actions";
import {
    GraduationCapIcon,
    UsersIcon,
    MailPlusIcon,
    BookOpenIcon,
    UploadIcon,
    CalendarCheckIcon,
    ClipboardCheckIcon,
    AlertTriangleIcon,
    HeartPulse,
    DollarSignIcon,
    FileTextIcon,
    BarChart3Icon,
} from "lucide-react";

export const TEACHER_ACTIONS: QuickAction[] = [
    {
        label: "Mark Attendance",
        description: "Record today's class attendance",
        href: "/attendance",
        icon: <CalendarCheckIcon className="size-4" />,
    },
    {
        label: "Record Assessment",
        description: "Enter scores for an assessment",
        href: "/assessments",
        icon: <ClipboardCheckIcon className="size-4" />,
    },
    {
        label: "Add Behaviour Note",
        description: "Record a commendation or incident",
        href: "/behavior",
        icon: <AlertTriangleIcon className="size-4" />,
    },
    {
        label: "My Classes",
        description: "View your class roster",
        href: "/classes",
        icon: <UsersIcon className="size-4" />,
    },
];

export const SCHOOL_ADMIN_ACTIONS: QuickAction[] = [
    {
        label: "Add Student",
        description: "Enrol a new student",
        href: "/students/new",
        icon: <GraduationCapIcon className="size-4" />,
    },
    {
        label: "Create Class",
        description: "Set up a new class",
        href: "/classes/add",
        icon: <UsersIcon className="size-4" />,
    },
    {
        label: "Invite Staff",
        description: "Send invitations to teachers",
        href: "/teachers/invitations",
        icon: <MailPlusIcon className="size-4" />,
    },
    {
        label: "Import Students",
        description: "Bulk import from file",
        href: "/students/import",
        icon: <UploadIcon className="size-4" />,
    },
    {
        label: "Setup Curriculum",
        description: "Learning areas and strands",
        href: "/curriculum",
        icon: <BookOpenIcon className="size-4" />,
    },
];

export const PARENT_ACTIONS: QuickAction[] = [
    {
        label: "View Reports",
        description: "Academic progress reports",
        href: "/reports",
        icon: <FileTextIcon className="size-4" />,
    },
    {
        label: "View Assessments",
        description: "Assessment scores and grades",
        href: "/assessments",
        icon: <BarChart3Icon className="size-4" />,
    },
    {
        label: "View Behaviour",
        description: "Commendations and incidents",
        href: "/behavior",
        icon: <AlertTriangleIcon className="size-4" />,
    },
];

export const NURSE_ACTIONS: QuickAction[] = [
    {
        label: "Record Incident",
        description: "Log a health or behaviour incident",
        href: "/health",
        icon: <HeartPulse className="size-4" />,
    },
    {
        label: "View Open Incidents",
        description: "Unresolved health records",
        href: "/health",
        icon: <AlertTriangleIcon className="size-4" />,
    },
];

export const FINANCE_ACTIONS: QuickAction[] = [
    {
        label: "Create Invoice",
        description: "Generate a new invoice",
        href: "/finance/invoices",
        icon: <DollarSignIcon className="size-4" />,
    },
    {
        label: "Fee Templates",
        description: "Manage fee structures",
        href: "/finance/fee-templates",
        icon: <FileTextIcon className="size-4" />,
    },
    {
        label: "View Invoices",
        description: "All invoices and payments",
        href: "/finance/invoices",
        icon: <BarChart3Icon className="size-4" />,
    },
];

export const SYSTEM_ADMIN_ACTIONS: QuickAction[] = [
    {
        label: "Manage Schools",
        description: "View and manage all schools",
        href: "/schools",
        icon: <GraduationCapIcon className="size-4" />,
    },
];
