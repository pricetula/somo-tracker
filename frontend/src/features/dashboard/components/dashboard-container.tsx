import { SummaryCards } from "./summary-cards";
import { NeedsAttention } from "./needs-attention";
import { TodayPanel } from "./today-panel";
import { ClassesTable } from "./classes-table";

export function DashboardContainer() {
    return (
        <div className="space-y-6 p-4">
            <header className="flex items-center justify-between">
                <h1 className="text-foreground text-xl font-medium">2026 · Term 3 · Active</h1>
                <div className="text-muted-foreground flex items-center gap-3 text-sm">
                    <a href="#" className="text-foreground">
                        + Add student
                    </a>
                    <a href="#" className="text-foreground">
                        📅 New ▾
                    </a>
                </div>
            </header>

            <SummaryCards />

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                <NeedsAttention />
                <TodayPanel />
            </div>

            <section className="border-t pt-4">
                <ClassesTable />
            </section>
        </div>
    );
}
