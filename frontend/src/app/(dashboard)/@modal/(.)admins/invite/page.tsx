"use client";
/**
 * Modal route — renders ImportOrchestrator wrapped in a Dialog shell.
 * Matches the intercepting route `@modal/(.)admins/invite`.
 */

import { ImportOrchestrator } from "@/features/import";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

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
export default function ImportModalPage() {
    return (
        <Dialog open>
            <DialogContent className="max-h-[85vh] max-w-4xl overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Import Data</DialogTitle>
                </DialogHeader>
                <ImportOrchestrator fieldDef={ADMIN_FIELDS} onMappedList={console.log} />
            </DialogContent>
        </Dialog>
    );
}
