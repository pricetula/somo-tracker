import { StudentSummaryCard } from "@/features/students";
import { ParentSummaryCard } from "@/features/guardians";
import { TeacherSummaryCard } from "@/features/teachers";

export function AdminDashboard() {
    return (
        <article>
            <header className="grid w-full max-w-5xl grid-cols-1 items-start gap-8 md:grid-cols-3 md:gap-6">
                <div className="border-r-0 pr-0 md:border-r md:border-dashed md:pr-6">
                    <StudentSummaryCard />
                </div>
                <div className="border-r-0 px-0 md:border-r md:border-dashed md:px-6">
                    <ParentSummaryCard />
                </div>
                <div className="pl-0 md:pl-6">
                    <TeacherSummaryCard />
                </div>
            </header>
        </article>
    );
}
