import Link from "next/link";
import { FileQuestion } from "lucide-react";

export default function DashboardNotFound() {
    return (
        <div className="flex min-h-100 flex-col items-center justify-center gap-4">
            <FileQuestion className="text-muted-foreground h-12 w-12" />
            <h1 className="text-2xl font-semibold tracking-tight">Page Not Found</h1>
            <p className="text-muted-foreground max-w-md text-center">
                The page you&apos;re looking for doesn&apos;t exist or has been moved.
            </p>
            <Link href="/">Go to Dashboard</Link>
        </div>
    );
}
