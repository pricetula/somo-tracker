import { Skeleton } from "@/components/ui/skeleton";

export function LessonTimelineSkeleton() {
    return (
        <div className="bg-background space-y-0">
            {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="relative flex gap-0 py-5 md:gap-2 lg:gap-4">
                    <aside className="sticky top-0 w-24 shrink-0 self-start px-2 pt-5 text-right md:w-32 md:px-3 lg:w-36">
                        <Skeleton className="mb-1 ml-auto h-4 w-16" />
                        <Skeleton className="ml-auto h-3 w-12" />
                    </aside>
                    <div className="flex w-10 shrink-0 flex-col items-center pt-5">
                        <Skeleton className="h-3 w-px" />
                        <Skeleton className="h-3 w-3 rounded-full" />
                        <Skeleton className="h-12 w-px" />
                    </div>
                    <div className="min-w-0 flex-1 space-y-3 pt-5 pr-4">
                        <div className="flex items-center gap-3">
                            <Skeleton className="h-6 w-40" />
                            <Skeleton className="h-5 w-16 rounded-full" />
                        </div>
                        <Skeleton className="h-4 w-3/4" />
                        <div className="flex gap-2">
                            <Skeleton className="h-5 w-20 rounded-full" />
                            <Skeleton className="h-5 w-14 rounded-full" />
                        </div>
                    </div>
                </div>
            ))}
        </div>
    );
}
