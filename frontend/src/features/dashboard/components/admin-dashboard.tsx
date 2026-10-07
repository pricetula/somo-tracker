import { StudentSummaryCard } from "@/features/students";
import { ParentSummaryCard } from "@/features/guardians";
import { TeacherSummaryCard } from "@/features/teachers";
import { UpcomingEventsWidget } from "./upcoming-events-widget";

export function AdminDashboard() {
    return (
        <article className="space-y-8">
            <header className="grid w-full max-w-5xl grid-cols-1 gap-6 md:grid-cols-3">
                <div className="border-r-0 border-b border-dashed pr-0 pb-4 md:border-r md:border-b-0 md:pr-6 md:pb-0">
                    <StudentSummaryCard />
                </div>
                <div className="border-r-0 border-b border-dashed pr-0 pb-4 md:border-r md:border-b-0 md:pr-6 md:pb-0">
                    <ParentSummaryCard />
                </div>
                <div className="pl-0 md:pl-6">
                    <TeacherSummaryCard />
                </div>
            </header>
        </article>
    );
}
