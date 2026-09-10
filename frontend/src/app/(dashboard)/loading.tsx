import { Skeleton } from "@/components/ui/skeleton";

/**
 * Dashboard loading skeleton.
 *
 * Shown while the ``DashboardAuthLayout`` server component is resolving.
 * Provides immediate feedback to the user that the page is loading.
 */
export default function DashboardLoading() {
    return (
        <div className="flex min-h-[64vh] flex-col items-center justify-center p-4">
            <Skeleton className="mb-4 h-4 w-64" />
            <Skeleton className="mb-2 h-4 w-48" />
            <Skeleton className="h-4 w-32" />
        </div>
    );
}
