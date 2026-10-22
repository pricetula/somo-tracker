import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function TodayPanel() {
    return (
        <Card className="h-full">
            <CardHeader className="pb-2">
                <CardTitle className="text-sm">Today · Tue 13 Oct</CardTitle>
            </CardHeader>
            <CardContent className="flex h-full flex-col justify-between">
                <div className="space-y-5">
                    {/* Attendance progress */}
                    <div>
                        <div className="text-muted-foreground mb-1.5 text-xs">
                            Attendance marked
                        </div>
                        <div className="mb-1.5 text-xl font-medium">18 of 24</div>
                        <div className="bg-muted flex w-full overflow-hidden rounded-full">
                            <div className="h-2 bg-emerald-600" style={{ width: "75%" }} />
                        </div>
                    </div>

                    {/* Live numbers */}
                    <div className="grid grid-cols-2 gap-3">
                        <div className="rounded border px-3 py-3">
                            <div className="text-muted-foreground text-xs">
                                Lessons in progress now
                            </div>
                            <div className="mt-0.5 text-xl font-medium">7</div>
                        </div>
                        <div className="rounded border px-3 py-3">
                            <div className="text-muted-foreground text-xs">Rooms in use now</div>
                            <div className="mt-0.5 text-xl font-medium">14 of 20</div>
                        </div>
                    </div>
                </div>

                {/* Next event */}
                <div className="bg-muted/40 rounded px-3 py-3 text-sm">
                    <div className="text-muted-foreground text-xs">Next event</div>
                    <div className="font-medium">Mid-term exams Mon</div>
                </div>
            </CardContent>
        </Card>
    );
}
