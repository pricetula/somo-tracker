import { StudentSummaryCard } from "@/features/students";
import { ParentSummaryCard } from "@/features/guardians";
import { TeacherSummaryCard } from "@/features/teachers";

export function AdminDashboard() {
    return (
        <article>
            <header className="flex w-full max-w-4xl flex-col gap-4 md:flex-row md:justify-between">
                <StudentSummaryCard />

                <ParentSummaryCard />

                <TeacherSummaryCard />
            </header>
        </article>
    );
}
