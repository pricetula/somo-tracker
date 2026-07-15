/**
 * Grading Scales settings page — manage grading scale profiles and their ranges.
 */

import { GradingScaleProfilesList } from "@/features/assessments";

export default function GradingScalesPage() {
    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-foreground text-2xl font-semibold">Grading Scales</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Manage grading scale profiles and their percentage-to-level mappings.
                </p>
            </div>
            <GradingScaleProfilesList />
        </div>
    );
}
