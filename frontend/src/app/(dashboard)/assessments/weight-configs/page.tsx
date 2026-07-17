/**
 * Weight Configurations page — manage assessment weight profiles.
 *
 * TODO: Implement weight config management UI once the backend API is ready.
 * This will allow schools to define how different assessment categories
 * (e.g., exams, assignments, projects) contribute to the final term grade.
 */

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function WeightConfigsPage() {
    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-foreground text-2xl font-semibold">Weight Configurations</h1>
                <p className="text-muted-foreground mt-1">
                    Define how different assessment types contribute to final term grades.
                </p>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Weight Profiles</CardTitle>
                    <CardDescription>
                        Configure weighting rules for assessments (e.g., exams 40%, assignments 30%,
                        projects 30%). Weight configs determine how individual assessment scores
                        roll up into term-level grades.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <p className="text-muted-foreground text-sm">
                        Weight configuration management is coming soon. Check back after the backend
                        API is available.
                    </p>
                </CardContent>
            </Card>
        </div>
    );
}
