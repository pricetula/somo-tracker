import { getAuthUser } from "@/lib/auth-server";
import { SchoolSettings } from "@/features/school";
import { redirect } from "next/navigation";

/**
 * Settings / Schools page — manage all schools in the organisation.
 * Only accessible by SYSTEM_ADMIN and SCHOOL_ADMIN roles.
 */
export default async function SettingsSchoolPage() {
    const user = await getAuthUser();

    // Only admins can manage schools
    if (!user || (user.role !== "SYSTEM_ADMIN" && user.role !== "SCHOOL_ADMIN")) {
        redirect("/");
    }

    return <SchoolSettings />;
}
