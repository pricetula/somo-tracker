"use client";

import { TermBanner } from "./term-banner";
import { EnrollmentMetrics } from "./enrollment-metrics";
import { AttendanceSnapshot } from "./attendance-snapshot";
import { HealthFlags } from "./health-flags";
import { PerformanceMatrix } from "./performance-matrix";
import { LearningAreaMastery } from "./learning-area-mastery";
import { StreamPerformance } from "./stream-variance";
import { YearOverYear } from "./year-over-year";
import { AttendanceScatter } from "./attendance-scatter";
import { Compliance } from "./compliance";
import { TeacherAudits } from "./teacher-audits";
import { Finance } from "./finance";
import { OperationalFeeds } from "./operational-feeds";
import {
    termBanner,
    enrollment,
    attendance,
    healthFlags,
    topPerformers,
    mostImproved,
    regressed,
    interventionList,
    masteryData,
    streamData,
    streams,
    yoyData,
    scatterData,
    complianceItems,
    evalCompliance,
    weightingIssues,
    absentees,
    finance,
    pendingUsers,
    medicalIncidents,
} from "./data";

export function CbcAdminDashboard() {
    return (
        <div className="mx-auto flex max-w-[1400px] flex-col gap-4 p-4">
            {/* ── Top Ribbon ── */}
            <div className="flex flex-col gap-4 md:flex-row">
                <div className="flex-1">
                    <TermBanner data={termBanner} />
                </div>
                <div className="flex-1">
                    <EnrollmentMetrics data={enrollment} />
                </div>
                <div className="flex-1">
                    <AttendanceSnapshot data={attendance} />
                </div>
                <div className="flex-1">
                    <HealthFlags data={healthFlags} />
                </div>
            </div>

            {/* ── Work Grid ── */}
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-[65%_35%]">
                {/* Left 65% */}
                <div className="flex flex-col gap-4">
                    <PerformanceMatrix
                        topPerformers={topPerformers}
                        mostImproved={mostImproved}
                        regressed={regressed}
                        interventionList={interventionList}
                    />

                    {/* 2×2 macro analytics grid */}
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        <LearningAreaMastery data={masteryData} />
                        <StreamPerformance data={streamData} streams={streams} />
                        <YearOverYear data={yoyData} hasPriorYear />
                        <AttendanceScatter data={scatterData} />
                    </div>
                </div>

                {/* Right 35% */}
                <div className="flex flex-col gap-4">
                    <Compliance items={complianceItems} />
                    <TeacherAudits
                        evalCompliance={evalCompliance}
                        weightingIssues={weightingIssues}
                        absentees={absentees}
                    />
                    <Finance data={finance} />
                    <OperationalFeeds
                        pendingUsers={pendingUsers}
                        medicalIncidents={medicalIncidents}
                    />
                </div>
            </div>
        </div>
    );
}
