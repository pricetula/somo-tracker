import { SummaryCards } from "./summary-cards";
import { NeedsAttention } from "./needs-attention";
import { DashboardHeader } from "./dashboard-header";
import { AttendanceKpi } from "./attendance-kpi";
import { UpcomingEvents } from "./upcoming-events";
import { ClassCoverageMini } from "./class-coverage-mini";

export function DashboardContainer() {
    return (
        <div>
            <DashboardHeader />

            <SummaryCards />

            <div className="mb-6 grid grid-cols-1 gap-3 md:grid-cols-2">
                <AttendanceKpi />
                <div className="flex items-center justify-between border p-3">
                    <div>
                        <div className="text-muted-foreground text-sm">Fees collected</div>
                        <div className="text-2xl">68%</div>
                        <div className="text-muted-foreground text-xs">38 students outstanding</div>
                    </div>
                    <div className="text-muted-foreground text-xs">spark soon</div>
                </div>
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
                <NeedsAttention />
                <UpcomingEvents />
                <ClassCoverageMini />
            </div>
        </div>
    );
}
