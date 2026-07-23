import { Skeleton } from "@/components/ui/skeleton";

export function TreeSkeleton() {
    return (
        <div className="space-y-4">
            <Skeleton className="h-8 w-64" />
            <Skeleton className="h-6 w-48" />
            {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="space-y-2">
                    <Skeleton className="h-8 w-full" />
                    <Skeleton className="ml-6 h-6 w-3/4" />
                    <Skeleton className="ml-12 h-5 w-1/2" />
                </div>
            ))}
        </div>
    );
}
