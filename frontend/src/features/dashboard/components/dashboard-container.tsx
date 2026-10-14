import { SummaryCards } from "./summary-cards";
import { NeedsAttention } from "./needs-attention";
import { TodayPanel } from "./today-panel";
import { DashboardHeader } from "./dashboard-header";
import { AttendanceKpi } from "./attendance-kpi";
import { FeeKpi } from "./fee-kpi";
import { UpcomingEvents } from "./upcoming-events";
import { ClassesTable } from "./classes-table";
import { TimetableHealth } from "./timetable-health";
import { HealthBehaviourPanel } from "./health-behaviour-panel";
import { MisbehaviourPanel } from "./misbehaviour-panel";

// Mock counts for state check
const mockStudents = 412;
const mockTeachers = 24;
const mockClasses = 28;

function isEstablished() {
    return mockStudents > 0 && mockTeachers > 0 && mockClasses > 0;
}

export function DashboardContainer() {
    const established = isEstablished();

    return (
        <div>
            <DashboardHeader />

            {/* Setup state link (visible when not fully set up) */}
            {!established && (
                <div className="text-muted-foreground mb-4 flex items-center gap-2 text-xs">
                    <span>Setup 3/5</span>
                    <a href="#" className="text-primary underline">
                        Continue setup
                    </a>
                </div>
            )}

            <SummaryCards />

            <div className="mb-6 grid grid-cols-1 gap-6 md:grid-cols-2">
                <AttendanceKpi />
                <FeeKpi />
            </div>

            {/* Two-panel row: Needs attention (2/3) + Today (1/3) */}
            {established ? (
                <div className="mb-6 grid grid-cols-1 gap-6 lg:grid-cols-3">
                    <div className="lg:col-span-2">
                        <NeedsAttention />
                    </div>
                    <div className="lg:col-span-1">
                        <TodayPanel />
                    </div>
                </div>
            ) : (
                <div className="mb-6 rounded border p-4">
                    <h3 className="mb-3 text-sm font-semibold">Get started</h3>
                    <ul className="space-y-2 text-sm">
                        {[
                            { label: "Create classes and streams", action: "Create" },
                            { label: "Add students", action: "Add" },
                            { label: "Invite teachers", action: "Invite" },
                            { label: "Set up the timetable", action: "Set up" },
                        ].map((step) => (
                            <li
                                key={step.label}
                                className="flex items-center justify-between border-b pb-2 last:border-0 last:pb-0"
                            >
                                <span className="text-muted-foreground">{step.label}</span>
                                <a href="#" className="text-primary text-xs font-medium underline">
                                    [{step.action}]
                                </a>
                            </li>
                        ))}
                    </ul>
                </div>
            )}

            {/* Full-width classes table */}
            <div className="mb-6">
                <ClassesTable />
            </div>

            {/* Bottom side-by-side: Timetable health + Events */}
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
                <TimetableHealth />
                <UpcomingEvents />
            </div>

            <div className="mt-6">
                <HealthBehaviourPanel />
            </div>

            <div className="mt-6">
                <MisbehaviourPanel />
            </div>
        </div>
    );
}
