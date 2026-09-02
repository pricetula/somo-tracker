import { getVerifiedRole } from "@/lib/auth-server";

interface AuthorizedByRoleProps {
    children: React.ReactNode;
    allowedRoles: string[];
}

export async function AuthorizedByRole({
    children,
    allowedRoles = ["TEACHER", "SCHOOL_ADMIN", "SYSTEM_ADMIN"],
}: AuthorizedByRoleProps) {
    if (!allowedRoles || allowedRoles.length === 0) {
        return null;
    }

    const role = await getVerifiedRole();

    if (!role) {
        return (
            <article>
                <p>Unable to verify your session. Please log in again.</p>
            </article>
        );
    }

    if (!allowedRoles.includes(role)) {
        return (
            <article>
                <p>You do not have access to this page.</p>
            </article>
        );
    }

    return children;
}
