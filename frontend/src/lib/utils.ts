import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}

export const userRole = {
    systemAdmin: "SYSTEM_ADMIN",
    schoolAdmin: "SCHOOL_ADMIN",
    teacher: "TEACHER",
    nurse: "NURSE",
    finance: "FINANCE",
    parent: "PARENT",
};

export type UserRole = "SYSTEM_ADMIN" | "SCHOOL_ADMIN" | "TEACHER" | "NURSE" | "FINANCE" | "PARENT";

export type UserRoleAccess = "CURRICULUM" | "STAFF" | "HEALTH";

export function isUserRoleAccess(role: string, roleAccess: UserRoleAccess): boolean {
    return (
        (roleAccess === "STAFF" &&
            ["SCHOOL_ADMIN", "TEACHER", "NURSE", "FINANCE"].includes(role)) ||
        (roleAccess === "CURRICULUM" && ["SCHOOL_ADMIN", "TEACHER"].includes(role)) ||
        (roleAccess === "HEALTH" && ["SCHOOL_ADMIN", "NURSE"].includes(role))
    );
}
