"use client";

import { UploadIcon, MailIcon } from "lucide-react";

export function ActivityIcon({ type }: { type: string }) {
    if (type === "staff_invite") {
        return <MailIcon className="size-4 shrink-0" />;
    }
    return <UploadIcon className="size-4 shrink-0" />;
}
