"use client";

import { useCallback, useState, useMemo, useSyncExternalStore } from "react";

import { DateRange } from "react-day-picker";
import { format, startOfYear, endOfYear } from "date-fns";
import { Trash2, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Calendar } from "@/components/ui/calendar";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from "@/components/ui/dialog";
import { YearSelect } from "@/components/shared/year-select";
import { useCreateAcademicPeriod } from "@/features/school/hooks/use-academic-period";

// Helper functions for client subscription
const emptySubscribe = () => () => {};
const getSnapshot = () => true;
const getServerSnapshot = () => false;

const initRange = { from: undefined, to: undefined };

function useIsMounted() {
    return useSyncExternalStore(emptySubscribe, getSnapshot, getServerSnapshot);
}

// ---- Types mirroring the Go structs ----
interface TermInput {
    name: string;
    start_date: string; // "2026-01-15"
    end_date: string;
}

interface AcademicPeriodRequest {
    year: number;
    terms: TermInput[];
}

function toISODate(date: Date): string {
    return format(date, "yyyy-MM-dd");
}

// props
interface CreateAcademicYearProps {
    onSuccess: () => void;
}

export function CreateAcademicYear({ onSuccess }: CreateAcademicYearProps) {
    const mutation = useCreateAcademicPeriod();
    const isMounted = useIsMounted();
    const currentDate = useMemo(() => new Date(), []);
    const currentYear = useMemo(() => currentDate.getFullYear(), [currentDate]);
    const [year, setYear] = useState<number>(currentYear);
    const [range, setRange] = useState<DateRange>(initRange);
    const [dialogOpen, setDialogOpen] = useState(false);
    const [termName, setTermName] = useState("");
    const [terms, setTerms] = useState<TermInput[]>([]);
    const payload: AcademicPeriodRequest = useMemo(() => ({ year, terms }), [year, terms]);
    const disableCreateButton = useMemo(() => !payload?.year || !payload?.terms?.length, [payload]);
    const { minDate, maxDate } = useMemo(() => {
        const d = new Date(year, 0);
        return { minDate: startOfYear(d), maxDate: endOfYear(d) };
    }, [year]);

    const handleRangeSelect = useCallback(
        (next: DateRange | undefined) => {
            const r = { ...range };
            if (!next?.from || !next?.to) return;
            if (!r?.from || (r?.from && r?.to)) {
                r.from = next.from;
                r.to = undefined;
            } else if (r?.from && !r?.to) {
                r.to = next.to;
            }
            setRange(r);
            // Open the dialog once both a start (from) and end (to) date are picked
            if (r?.from && r?.to) {
                setDialogOpen(true);
            }
        },
        [range]
    );
    const resetSelection = useCallback(() => {
        setRange(initRange);
        setTermName("");
    }, []);
    const handleCancel = useCallback(() => {
        setDialogOpen(false);
        resetSelection();
    }, [resetSelection]);
    const handleSave = useCallback(() => {
        if (!termName.trim() || !range?.from || !range?.to) return;

        setTerms((prev) => [
            ...prev,
            {
                name: termName.trim(),
                start_date: toISODate(range.from as Date),
                end_date: toISODate(range.to as Date),
            },
        ]);

        setDialogOpen(false);
        resetSelection();
    }, [termName, range, resetSelection]);
    const removeTerm = useCallback((index: number) => {
        setTerms((prev) => prev.filter((_, i) => i !== index));
    }, []);

    return (
        <div>
            <div>
                <h1 className="text-2xl font-semibold">Academic period setup</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Choose a year, select a date range for each term, name and save it.
                </p>
            </div>

            <div className="flex items-start gap-4">
                <div className="space-y-4">
                    <Card>
                        <CardHeader>
                            <CardTitle className="text-base">Academic year</CardTitle>
                        </CardHeader>
                        <CardContent>
                            <YearSelect
                                year={year}
                                onSelect={(y) => {
                                    setYear(y);
                                    setTerms([]);
                                    setRange(initRange);
                                }}
                            />
                        </CardContent>
                    </Card>

                    <Card>
                        <CardHeader>
                            <CardTitle className="text-base">Term dates</CardTitle>
                        </CardHeader>
                        <CardContent className="flex flex-col items-center">
                            {isMounted && (
                                <Calendar
                                    mode="range"
                                    selected={range}
                                    onSelect={handleRangeSelect}
                                    numberOfMonths={2}
                                    className="rounded-lg border"
                                    startMonth={minDate}
                                    endMonth={maxDate}
                                />
                            )}
                            <p className="text-muted-foreground mt-3 self-start text-xs">
                                {range?.from
                                    ? `${format(range.from, "MMM d, yyyy")}${
                                          range.to
                                              ? ` → ${format(range.to, "MMM d, yyyy")}`
                                              : " → select an end date"
                                      }`
                                    : "Click a start date, then an end date."}
                            </p>
                        </CardContent>
                    </Card>
                </div>

                <Card className="w-70">
                    <CardHeader>
                        <CardTitle className="text-base">Terms for {year}</CardTitle>
                    </CardHeader>
                    <CardContent>
                        {terms.length === 0 ? (
                            <p className="text-muted-foreground text-sm italic">
                                No terms added yet.
                            </p>
                        ) : (
                            <ul className="space-y-2">
                                {terms.map((t, i) => (
                                    <li
                                        key={i}
                                        className="flex items-start justify-between gap-2 rounded-md border px-3 py-2"
                                    >
                                        <div>
                                            <p className="text-sm font-medium">{t.name}</p>
                                            <p className="text-muted-foreground text-xs">
                                                {t.start_date} → {t.end_date}
                                            </p>
                                        </div>
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            className="h-7 w-7"
                                            onClick={() => removeTerm(i)}
                                            aria-label={`Remove ${t.name}`}
                                        >
                                            <Trash2 className="h-3.5 w-3.5" />
                                        </Button>
                                    </li>
                                ))}
                            </ul>
                        )}
                    </CardContent>
                    <CardFooter>
                        <Button
                            disabled={disableCreateButton || mutation.isPending}
                            onClick={() =>
                                mutation.mutate(payload, { onSuccess: () => onSuccess() })
                            }
                        >
                            {mutation.isPending ? "Creating..." : "Create"}
                        </Button>
                    </CardFooter>
                </Card>
            </div>

            {/* Dialog: non-dismissable except via Cancel/Save */}
            <Dialog open={dialogOpen} onOpenChange={() => {}}>
                <DialogContent className="sm:max-w-sm [&>button]:hidden" showCloseButton={false}>
                    <DialogHeader>
                        <DialogTitle>Name this term</DialogTitle>
                    </DialogHeader>

                    <p className="-mt-2 text-sm text-amber-600">
                        {range?.from && format(range.from, "MMM d, yyyy")}
                        {" → "}
                        {range?.to && format(range.to, "MMM d, yyyy")}
                    </p>

                    <div className="space-y-1.5">
                        <Label htmlFor="term-name">Term name</Label>
                        <Input
                            id="term-name"
                            autoFocus
                            value={termName}
                            onChange={(e) => setTermName(e.target.value)}
                            placeholder="Term 1"
                            onKeyDown={(e) => {
                                if (e.key === "Enter") handleSave();
                            }}
                        />
                    </div>

                    <DialogFooter>
                        <Button variant="outline" onClick={handleCancel}>
                            Cancel
                        </Button>
                        <Button onClick={handleSave} disabled={!termName.trim()}>
                            <Plus className="mr-1 h-3.5 w-3.5" />
                            Save term
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
