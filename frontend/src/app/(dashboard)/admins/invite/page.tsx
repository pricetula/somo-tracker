"use client";
/**
 * Full-page import route.
 * Renders ImportOrchestrator directly (no Dialog shell).
 */

import { ImportOrchestrator } from "@/features/import";
// Sample admin import schema
const ADMIN_FIELDS = [
    {
        key: "full_name",
        label: "Full name",
        aliases: ["name", "first name", "first_name", "full name"],
        required: true,
    },
    {
        key: "email",
        label: "Email",
        aliases: ["e-mail", "mail", "email address"],
        required: true,
        validator: (value: unknown) => {
            if (typeof value !== "string" || !value.includes("@")) return "Invalid email";
            return null;
        },
    },
    {
        key: "role",
        label: "Role",
        aliases: ["user role", "permission", "access"],
        required: true,
    },
    {
        key: "school_slug",
        label: "School slug",
        aliases: ["school", "academy", "institution"],
        required: false,
    },
];
export default function ImportPage() {
    return <ImportOrchestrator fieldDef={ADMIN_FIELDS} onMappedList={console.log} />;
}
