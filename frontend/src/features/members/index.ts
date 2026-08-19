/**
 * Members feature — staff, parent, and student member management.
 *
 * Import only from this barrel, never from internal paths.
 */

// Listing pages
export { AdminsPage } from "./components/admins-page";
export { TeachersPage } from "./components/teachers-page";
export { ParentsPage } from "./components/parents-page";
export { NursesPage } from "./components/nurses-page";
export { FinancePage } from "./components/finance-page";
export { StudentsPage } from "./components/students-page";

// Full-page invite forms — rendered by `/<role>/add` page routes on
// hard navigation / refresh.
export { AdminsInvitePage } from "./components/admins-invite-page";
export { TeachersInvitePage } from "./components/teachers-invite-page";
export { ParentsInvitePage } from "./components/parents-invite-page";
export { NursesInvitePage } from "./components/nurses-invite-page";
export { FinanceInvitePage } from "./components/finance-invite-page";

// Dialog invite forms — rendered by the `@modal` parallel slot on
// client-side navigation.
export { AdminsInviteDialog } from "./components/admins-invite-dialog";
export { TeachersInviteDialog } from "./components/teachers-invite-dialog";
export { ParentsInviteDialog } from "./components/parents-invite-dialog";
export { NursesInviteDialog } from "./components/nurses-invite-dialog";
export { FinanceInviteDialog } from "./components/finance-invite-dialog";
