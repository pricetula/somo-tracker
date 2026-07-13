/**
 * Student Detail Page — Full page render for /students/:id.
 *
 * On hard refresh, this renders the full page.
 * When client-navigated from the class roster, it is intercepted
 * by @modal/(.)students/[id] and rendered as a side sheet instead.
 */

import { Construction } from "lucide-react";

interface Props {
    params: Promise<{ id: string }>;
}

export default async function StudentDetailPage({ params }: Props) {
    await params; // resolve route params

    return (
        <div className="flex flex-1 flex-col items-center justify-center px-6 py-24">
            <Construction className="text-muted-foreground mb-4 h-12 w-12" />
            <h1 className="text-foreground text-2xl font-semibold tracking-tight">
                Student Profile
            </h1>
            <p className="text-muted-foreground mt-2 text-center">
                The student detail page is coming soon.
            </p>
        </div>
    );
}
