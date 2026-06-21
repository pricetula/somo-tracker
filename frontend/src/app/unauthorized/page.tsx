import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ShieldAlert } from "lucide-react";

/**
 * Unauthorized page — shown when a user is authenticated but does not have
 * permission to access the requested route.
 *
 * The proxy redirects authenticated users here when their role is not in the
 * allowed routes for the path they tried to visit.
 *
 * This is intentionally a simple standalone page without dashboard chrome,
 * since the user doesn't have dashboard access.
 */
export default function UnauthorizedPage() {
    return (
        <div className="bg-background flex min-h-screen flex-col items-center justify-center">
            <div className="mx-auto flex max-w-md flex-col items-center gap-6 px-4 text-center">
                <div className="bg-destructive/10 rounded-full p-4">
                    <ShieldAlert className="text-destructive h-12 w-12" />
                </div>

                <div className="space-y-2">
                    <h1 className="text-3xl font-bold tracking-tight">Access Denied</h1>
                    <p className="text-muted-foreground text-base leading-relaxed">
                        You don&apos;t have permission to access this page. If you believe this is a
                        mistake, please contact your school administrator.
                    </p>
                </div>

                <Button asChild>
                    <Link href="/logout">Sign out and try again</Link>
                </Button>
            </div>
        </div>
    );
}
