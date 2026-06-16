"use client";

import * as React from "react";
import { useForm, useFieldArray, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { format } from "date-fns";
import {
  CheckCircle2,
  Flag,
  Loader2,
  Plus,
  Trash2,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { DatePicker } from "@/features/calendar/components/date-picker";
import { useSaveAcademicCalendar } from "@/features/calendar/hooks/use-academic-calendar";

// ─── Zod Schema ───────────────────────────────────────────────────────────

const periodSchema = z.object({
  name: z.string().min(1, "Period name is required"),
  startDate: z.date({ required_error: "Start date is required" }),
  endDate: z.date({ required_error: "End date is required" }),
  isFinal: z.boolean(),
});

const formSchema = z
  .object({
    year: z
      .number({ required_error: "Year is required" })
      .int()
      .min(2020, "Year must be 2020 or later")
      .max(2100, "Year must be 2100 or earlier"),
    periods: z.array(periodSchema).min(1, "At least one period is required"),
  })
  .refine(
    (data) => {
      // Ensure each period has start < end
      return data.periods.every(
        (p) => p.startDate && p.endDate && p.endDate > p.startDate,
      );
    },
    { message: "Each period's end date must be after its start date" },
  )
  .refine(
    (data) => {
      // Ensure sequential periods don't overlap
      for (let i = 1; i < data.periods.length; i++) {
        const prev = data.periods[i - 1];
        const curr = data.periods[i];
        if (prev.endDate && curr.startDate && curr.startDate <= prev.endDate) {
          return false;
        }
      }
      return true;
    },
    { message: "Periods must not overlap" },
  );

type FormValues = z.infer<typeof formSchema>;

// ─── Default Period Names ─────────────────────────────────────────────────

const DEFAULT_PERIODS = ["Term 1", "Term 2", "Term 3"];

// ─── Props ────────────────────────────────────────────────────────────────

interface AcademicCalendarFormProps {
  /** When true, the form slides up after successful save */
  onSuccess?: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function AcademicCalendarForm({ onSuccess }: AcademicCalendarFormProps) {
  const [showSuccess, setShowSuccess] = React.useState(false);
  const saveMutation = useSaveAcademicCalendar();

  const currentYear = new Date().getFullYear();

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      year: currentYear,
      periods: DEFAULT_PERIODS.map((name, i) => ({
        name,
        startDate: undefined,
        endDate: undefined,
        isFinal: i === DEFAULT_PERIODS.length - 1, // last term final by default
      })),
    },
    mode: "onChange",
  });

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "periods",
  });

  const watchedPeriods = form.watch("periods");

  // Determine if the form is valid for submission
  const isFormValid = form.formState.isValid;

  // Compute min/max date boundaries based on selected year
  const selectedYear = form.watch("year");
  const yearStart = React.useMemo(
    () => (selectedYear ? new Date(selectedYear, 0, 1) : undefined),
    [selectedYear],
  );
  const yearEnd = React.useMemo(
    () => (selectedYear ? new Date(selectedYear, 11, 31) : undefined),
    [selectedYear],
  );

  // ─── Submission Handler ───────────────────────────────────────────────

  async function onSubmit(data: FormValues) {
    // The schema ensures all dates are defined at this point
    const payload = {
      year: data.year,
      periods: data.periods.map((period) => ({
        name: period.name,
        start_date: format(period.startDate!, "yyyy-MM-dd"),
        end_date: format(period.endDate!, "yyyy-MM-dd"),
        is_final: period.isFinal, // visible toggle per row
      })),
    };

    try {
      await saveMutation.mutateAsync(payload);
      // ─── Success Resolution ───────────────────────────────────────
      setShowSuccess(true);
      // Fade out and notify parent after a brief animation
      setTimeout(() => {
        onSuccess?.();
      }, 1500);
    } catch {
      // Toast handles error display
    }
  }

  // ─── Success State ──────────────────────────────────────────────────

  if (showSuccess) {
    return (
      <div className="flex items-center justify-center py-12 transition-all duration-500">
        <div className="flex flex-col items-center gap-4 text-center">
          <div className="animate-in zoom-in-50 fade-in duration-500">
            <CheckCircle2 className="h-16 w-16 text-emerald-500" />
          </div>
          <p className="text-lg font-medium text-emerald-700 dark:text-emerald-400">
            Calendar activated successfully!
          </p>
        </div>
      </div>
    );
  }

  // ─── Form State ─────────────────────────────────────────────────────

  const isSubmitting = saveMutation.isPending;

  return (
    <div
      className={`rounded-2xl border bg-card p-6 shadow-sm transition-all duration-300 ${
        isSubmitting ? "pointer-events-none opacity-60" : ""
      }`}
    >
      <div className="mb-6 flex items-center gap-2">
        <span className="text-xl" role="img" aria-label="calendar">
          🗓️
        </span>
        <h2 className="text-lg font-semibold">Set Up Academic Calendar</h2>
      </div>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
          {/* ── Year Input ───────────────────────────────────────────── */}
          <FormField
            control={form.control}
            name="year"
            render={({ field }) => (
              <FormItem className="mx-auto max-w-xs text-center">
                <FormLabel>Select Academic Year</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    min={2020}
                    max={2100}
                    className="text-center text-lg"
                    disabled={isSubmitting}
                    {...field}
                    onChange={(e) => field.onChange(Number(e.target.value))}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          {/* ── Periods Section ──────────────────────────────────────── */}
          <div>
            <div className="mb-3 flex items-center justify-between">
              <h3 className="text-base font-medium text-foreground">Define Academic Periods</h3>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={isSubmitting}
                onClick={() =>
                  append({ name: "", startDate: undefined as unknown as Date, endDate: undefined as unknown as Date, isFinal: false })
                }
              >
                <Plus className="mr-1 h-4 w-4" />
                Add Period Row
              </Button>
            </div>

            <div className="space-y-4">
              {fields.map((field, index) => {
                const period = watchedPeriods[index];

                // Compute min/max for this row's date pickers
                // Row start: Jan 1 of selected year, or prev period's end + 1 day
                let minStart: Date | undefined = yearStart;
                if (index > 0 && watchedPeriods[index - 1]?.endDate) {
                  const prevEnd = watchedPeriods[index - 1].endDate!;
                  const nextDay = new Date(prevEnd);
                  nextDay.setDate(nextDay.getDate() + 1);
                  minStart = nextDay > (yearStart ?? nextDay) ? nextDay : yearStart;
                }

                // Row end: must be after this row's start
                const minEnd = period?.startDate
                  ? (() => {
                      const nextDay = new Date(period.startDate);
                      nextDay.setDate(nextDay.getDate() + 1);
                      return nextDay;
                    })()
                  : minStart;

                return (
                  <div
                    key={field.id}
                    className="flex flex-wrap items-end gap-3 rounded-lg border bg-muted/30 p-4"
                  >
                    {/* Period Name */}
                    <FormField
                      control={form.control}
                      name={`periods.${index}.name`}
                      render={({ field }) => (
                        <FormItem className="min-w-[120px] flex-1">
                          <FormLabel className="sr-only">Name</FormLabel>
                          <FormControl>
                            <Input
                              placeholder={`Term ${index + 1}`}
                              disabled={isSubmitting}
                              {...field}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    {/* Start Date */}
                    <FormField
                      control={form.control}
                      name={`periods.${index}.startDate`}
                      render={() => (
                        <FormItem className="min-w-[160px] flex-1">
                          <FormLabel className="sr-only">Start Date</FormLabel>
                          <FormControl>
                            <Controller
                              control={form.control}
                              name={`periods.${index}.startDate`}
                              render={({ field: dateField }) => (
                                <DatePicker
                                  value={dateField.value}
                                  onChange={dateField.onChange}
                                  disabled={isSubmitting}
                                  minDate={minStart}
                                  maxDate={yearEnd}
                                  placeholder="Start date"
                                />
                              )}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    {/* End Date */}
                    <FormField
                      control={form.control}
                      name={`periods.${index}.endDate`}
                      render={() => (
                        <FormItem className="min-w-[160px] flex-1">
                          <FormLabel className="sr-only">End Date</FormLabel>
                          <FormControl>
                            <Controller
                              control={form.control}
                              name={`periods.${index}.endDate`}
                              render={({ field: dateField }) => (
                                <DatePicker
                                  value={dateField.value}
                                  onChange={dateField.onChange}
                                  disabled={isSubmitting}
                                  minDate={minEnd}
                                  maxDate={yearEnd}
                                  placeholder="End date"
                                />
                              )}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />

                    {/* Is Final Toggle */}
                    <FormField
                      control={form.control}
                      name={`periods.${index}.isFinal`}
                      render={({ field }) => (
                        <FormItem className="shrink-0">
                          <FormLabel className="sr-only">Final period</FormLabel>
                          <FormControl>
                            <Button
                              type="button"
                              variant={field.value ? "default" : "outline"}
                              size="sm"
                              disabled={isSubmitting}
                              className="gap-1.5 whitespace-nowrap"
                              onClick={() => {
                                // Radio-group behavior: set this one to true, all others to false
                                const periods = form.getValues("periods");
                                periods.forEach((_, i) => {
                                  form.setValue(`periods.${i}.isFinal`, i === index);
                                });
                                form.trigger(`periods.${index}.isFinal`);
                              }}
                            >
                              <Flag className={`h-3.5 w-3.5 ${field.value ? "fill-current" : ""}`} />
                              {field.value ? "Final" : "Set Final"}
                            </Button>
                          </FormControl>
                        </FormItem>
                      )}
                    />

                    {/* Remove Button */}
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="shrink-0 text-muted-foreground hover:text-destructive"
                      disabled={fields.length <= 1 || isSubmitting}
                      onClick={() => {
                        // If removing the final period, reassign final to the last remaining
                        const currentFinal = form.getValues("periods")[index]?.isFinal;
                        remove(index);
                        if (currentFinal) {
                          const remaining = form.getValues("periods");
                          // After remove, the array has shifted
                          setTimeout(() => {
                            const updated = form.getValues("periods");
                            if (updated.length > 0) {
                              form.setValue(`periods.${updated.length - 1}.isFinal`, true);
                            }
                          }, 0);
                        }
                      }}
                    >
                      <Trash2 className="h-4 w-4" />
                      <span className="sr-only">Remove period</span>
                    </Button>
                  </div>
                );
              })}
            </div>
            {form.formState.errors.periods?.root && (
              <p className="mt-2 text-sm text-destructive font-medium">
                {form.formState.errors.periods.root.message}
              </p>
            )}
          </div>

          {/* ── Action Row ─────────────────────────────────────────────── */}
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <Button
              type="submit"
              className="flex-1 sm:flex-none"
              disabled={!isFormValid || isSubmitting}
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Saving...
                </>
              ) : (
                "Save & Activate Calendar"
              )}
            </Button>

            <Button
              type="button"
              variant="outline"
              disabled={isSubmitting}
              onClick={() => {
                // Fill demo data: Kenyan CBC default terms
                const demoStart = new Date(selectedYear, 0, 6); // Jan 6
                const demoPeriods = [
                  { start: new Date(demoStart), end: new Date(selectedYear, 3, 11) }, // Jan 6 – Apr 11
                  { start: new Date(selectedYear, 4, 5), end: new Date(selectedYear, 7, 15) }, // May 5 – Aug 15
                  { start: new Date(selectedYear, 8, 1), end: new Date(selectedYear, 10, 28) }, // Sep 1 – Nov 28
                ];
                demoPeriods.forEach((p, i) => {
                  form.setValue(`periods.${i}.startDate`, p.start);
                  form.setValue(`periods.${i}.endDate`, p.end);
                  form.setValue(`periods.${i}.name`, `Term ${i + 1}`);
                  form.setValue(`periods.${i}.isFinal`, i === demoPeriods.length - 1);
                });
                form.trigger();
              }}
            >
              Fill with Sample CBC Data
            </Button>
          </div>
        </form>
      </Form>
    </div>
  );
}
