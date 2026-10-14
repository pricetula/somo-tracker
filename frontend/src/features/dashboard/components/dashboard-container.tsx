import { SummaryCards } from "./summary-cards";
import { NeedsAttention } from "./needs-attention";
import { DashboardHeader } from "./dashboard-header";
import { AttendanceKpi } from "./attendance-kpi";
import { FeeKpi } from "./fee-kpi";
import { UpcomingEvents } from "./upcoming-events";
import { ClassCoverageMini } from "./class-coverage-mini";

export function DashboardContainer() {
    return (
        <div>
            <DashboardHeader />

            <SummaryCards />

            <div className="mb-6 grid grid-cols-1 md:grid-cols-2">
                <AttendanceKpi />
                <FeeKpi />
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
                <NeedsAttention />
                <UpcomingEvents />
                <ClassCoverageMini />
            </div>
        </div>
    );
}
