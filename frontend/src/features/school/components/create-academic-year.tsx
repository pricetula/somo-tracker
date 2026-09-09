"use client";

import { useState } from "react";
import { DateRange } from "react-day-picker";
import { format } from "date-fns";
import { Copy, Check, Trash2, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
    const currentYear = new Date().getFullYear();

    const [year, setYear] = useState<number>(currentYear);
    const [range, setRange] = useState<DateRange | undefined>(undefined);

    const [dialogOpen, setDialogOpen] = useState(false);
    const [termName, setTermName] = useState("");

    const [terms, setTerms] = useState<TermInput[]>([]);
    const [copied, setCopied] = useState(false);

    function handleRangeSelect(next: DateRange | undefined) {
        setRange(next);
        // Open the dialog once both a start (from) and end (to) date are picked
        if (next?.from && next?.to) {
            setDialogOpen(true);
        }
    }

    function resetSelection() {
        setRange(undefined);
        setTermName("");
    }

    function handleCancel() {
        setDialogOpen(false);
        resetSelection();
    }

    function handleSave() {
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
    }

    function removeTerm(index: number) {
        setTerms((prev) => prev.filter((_, i) => i !== index));
    }

    const payload: AcademicPeriodRequest = { year, terms };
    const payloadJson = JSON.stringify(payload, null, 2);

    function copyJson() {
        navigator.clipboard?.writeText(payloadJson);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
    }

    return (
        <div className="mx-auto max-w-4xl space-y-6 p-6">
            <div>
                <h1 className="text-2xl font-semibold">Academic period setup</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Choose a year, select a date range for each term, then name and save it.
                </p>
            </div>

            <div className="grid grid-cols-1 gap-6 md:grid-cols-5">
                {/* Left: year + calendar */}
                <div className="space-y-4 md:col-span-3">
                    <Card>
                        <CardHeader>
                            <CardTitle className="text-base">Academic year</CardTitle>
                        </CardHeader>
                        <CardContent>
                            <YearSelect year={year} onSelect={setYear} />
                            <p className="text-muted-foreground mt-2 text-xs">
                                Years beyond {currentYear} aren&apos;t offered.
                            </p>
                        </CardContent>
                    </Card>

                    <Card>
                        <CardHeader>
                            <CardTitle className="text-base">Term dates</CardTitle>
                        </CardHeader>
                        <CardContent className="flex flex-col items-center">
                            <Calendar
                                mode="range"
                                selected={range}
                                onSelect={handleRangeSelect}
                                numberOfMonths={1}
                                defaultMonth={new Date(year, 0)}
                                className="rounded-md border"
                            />
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

                {/* Right: terms list + JSON preview */}
                <div className="space-y-4 md:col-span-2">
                    <Card>
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
                    </Card>

                    <Card className="border-zinc-800 bg-zinc-950 text-zinc-100">
                        <CardHeader className="flex flex-row items-center justify-between py-3">
                            <CardTitle className="font-mono text-xs text-zinc-400">
                                AcademicPeriodRequest
                            </CardTitle>
                            <Button
                                variant="ghost"
                                size="sm"
                                className="h-7 text-zinc-300 hover:bg-zinc-800 hover:text-white"
                                onClick={copyJson}
                            >
                                {copied ? (
                                    <Check className="mr-1 h-3.5 w-3.5" />
                                ) : (
                                    <Copy className="mr-1 h-3.5 w-3.5" />
                                )}
                                {copied ? "Copied" : "Copy"}
                            </Button>
                        </CardHeader>
                        <CardContent>
                            <pre className="overflow-x-auto font-mono text-xs whitespace-pre">
                                {payloadJson}
                            </pre>
                        </CardContent>
                    </Card>
                </div>
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
