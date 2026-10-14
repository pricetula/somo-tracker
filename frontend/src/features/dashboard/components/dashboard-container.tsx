import { SummaryCards } from "./summary-cards";
import { NeedsAttention } from "./needs-attention";
import { TodayPanel } from "./today-panel";
import { ClassesTable } from "./classes-table";
import { DashboardHeader } from "./dashboard-header";

export function DashboardContainer() {
    return (
        <div>
            <DashboardHeader />

            <SummaryCards />

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                <NeedsAttention />
                <TodayPanel />
            </div>

            <ClassesTable />
        </div>
    );
}
