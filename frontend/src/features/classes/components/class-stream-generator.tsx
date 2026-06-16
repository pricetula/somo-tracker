"use client";

import * as React from "react";
import { CheckCircle2, Loader2, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { CBE_GRADE_TIERS } from "@/features/classes/types";
import { useGenerateClasses } from "@/features/classes/hooks/use-classes";

// ─── Props ────────────────────────────────────────────────────────────────

interface ClassStreamGeneratorProps {
  /** Called after successful generation, after the success animation. */
  onSuccess?: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function ClassStreamGenerator({ onSuccess }: ClassStreamGeneratorProps) {
  const [streams, setStreams] = React.useState<string[]>([]);
  const [inputValue, setInputValue] = React.useState("");
  const [showSuccess, setShowSuccess] = React.useState(false);
  const [fadeOut, setFadeOut] = React.useState(false);

  const inputRef = React.useRef<HTMLInputElement>(null);
  const generateMutation = useGenerateClasses();

  // Refs for container element for slide-up
  const containerRef = React.useRef<HTMLDivElement>(null);

  // ─── Tag Input Handlers ─────────────────────────────────────────────

  /** Commit the current input value as a new stream tag. */
  function addStreamTag(value: string) {
    const trimmed = value.trim();
    if (!trimmed) return;

    // Prevent duplicates (case-insensitive)
    if (streams.some((s) => s.toLowerCase() === trimmed.toLowerCase())) {
      return;
    }

    setStreams((prev) => [...prev, trimmed]);
    setInputValue("");
  }

  /** Remove a stream tag by index. */
  function removeStreamTag(index: number) {
    setStreams((prev) => prev.filter((_, i) => i !== index));
  }

  /** Handle key events on the input field. */
  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault();
      addStreamTag(inputValue);
    }
    // Backspace on empty input removes the last tag
    if (e.key === "Backspace" && inputValue === "" && streams.length > 0) {
      removeStreamTag(streams.length - 1);
    }
  }

  /** Handle paste: split on comma/newline/tab and add all. */
  function handlePaste(e: React.ClipboardEvent<HTMLInputElement>) {
    e.preventDefault();
    const pasted = e.clipboardData.getData("text");
    const items = pasted.split(/[,\n\t]+/).map((s) => s.trim()).filter(Boolean);
    if (items.length > 0) {
      setStreams((prev) => {
        const combined = [...prev];
        for (const item of items) {
          if (!combined.some((s) => s.toLowerCase() === item.toLowerCase())) {
            combined.push(item);
          }
        }
        return combined;
      });
    }
  }

  // ─── Real-Time Cross-Multiplication Preview ─────────────────────────

  /** Compute the cross product: CBE_GRADE_TIERS × streams. */
  const previewGrid = React.useMemo(() => {
    if (streams.length === 0) return [];
    const grid: string[] = [];
    for (const grade of CBE_GRADE_TIERS) {
      for (const stream of streams) {
        grid.push(`${grade} ${stream}`);
      }
    }
    return grid;
  }, [streams]);

  // ─── Button Guard ───────────────────────────────────────────────────

  const canSubmit = streams.length > 0 && !generateMutation.isPending;

  // ─── Submission Handler ─────────────────────────────────────────────

  async function handleSubmit() {
    if (!canSubmit) return;

    try {
      await generateMutation.mutateAsync({ streams });

      // ─── Success Resolution ────────────────────────────────────
      setFadeOut(true);

      // Brief delay for fade animation, then show checkmark
      setTimeout(() => {
        setShowSuccess(true);
      }, 400);

      // After checkmark, slide-up and notify parent
      setTimeout(() => {
        onSuccess?.();
      }, 2000);
    } catch {
      // Toast handles error display
    }
  }

  // ─── Success State ─────────────────────────────────────────────────

  if (showSuccess) {
    return (
      <div className="flex items-center justify-center py-12 transition-all duration-500">
        <div className="flex flex-col items-center gap-4 text-center">
          <div className="animate-in zoom-in-50 fade-in duration-500">
            <CheckCircle2 className="h-16 w-16 text-emerald-500" />
          </div>
          <p className="text-lg font-medium text-emerald-700 dark:text-emerald-400">
            Classrooms generated successfully!
          </p>
        </div>
      </div>
    );
  }

  // ─── Form State ────────────────────────────────────────────────────

  const isSubmitting = generateMutation.isPending;

  return (
    <div
      ref={containerRef}
      className={`rounded-2xl border bg-card p-6 shadow-sm transition-all duration-500 ${
        isSubmitting ? "pointer-events-none opacity-60" : ""
      } ${fadeOut ? "opacity-0 scale-95" : ""}`}
    >
      {/* Header */}
      <div className="mb-6 flex items-center gap-2">
        <span className="text-xl" role="img" aria-label="school">
          🏫
        </span>
        <h2 className="text-lg font-semibold">
          Step 2: Establish Your Classes &amp; Streams
        </h2>
      </div>

      {/* ── Stream Tag Input ─────────────────────────────────────── */}
      <div className="mb-4">
        <label className="mb-2 block text-sm font-medium text-foreground">
          Define your Streams / Sections
        </label>
        <div className="flex flex-wrap items-center gap-2 rounded-lg border border-input bg-background px-3 py-2 focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-1">
          {streams.map((stream, index) => (
            <span
              key={stream}
              className="inline-flex items-center gap-1 rounded-md bg-primary/10 px-2.5 py-1 text-sm font-medium text-primary"
            >
              {stream}
              <button
                type="button"
                onClick={() => removeStreamTag(index)}
                className="inline-flex items-center rounded-sm p-0.5 text-primary/60 hover:text-primary focus:outline-none"
                disabled={isSubmitting}
                aria-label={`Remove ${stream}`}
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </span>
          ))}
          <input
            ref={inputRef}
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            onPaste={handlePaste}
            aria-label="Stream name input"
            placeholder={
              streams.length === 0
                ? "Type a stream name (e.g. East) and press Enter..."
                : "Add another stream..."
            }
            disabled={isSubmitting}
            className="min-w-[200px] flex-1 border-0 bg-transparent p-0 text-sm outline-none placeholder:text-muted-foreground"
          />
        </div>
        <p className="mt-1.5 text-xs text-muted-foreground">
          Press Enter to add a stream. You can also paste multiple names (comma-separated).
        </p>
      </div>

      {/* ── Live Preview Grid ────────────────────────────────────── */}
      {previewGrid.length > 0 && (
        <div className="mb-6">
          <div className="mb-2 flex items-center gap-2">
            <span className="text-sm font-medium text-foreground">
              🔍 Preview of classrooms to be generated automatically:
            </span>
            <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
              {previewGrid.length} total
            </span>
          </div>
          <div className="grid grid-cols-2 gap-2 rounded-xl border bg-muted/30 p-4 sm:grid-cols-3 md:grid-cols-4">
            {previewGrid.map((name) => (
              <div
                key={name}
                data-testid="preview-cell"
                className="rounded-lg border bg-background px-3 py-2 text-center text-sm font-medium text-foreground shadow-sm transition-colors hover:border-primary/30"
              >
                {name}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Empty state when no streams are added yet */}
      {previewGrid.length === 0 && (
        <div className="mb-6 flex items-center justify-center rounded-xl border border-dashed bg-muted/20 p-8">
          <p className="text-sm text-muted-foreground">
            Add a stream above to see a live preview of your classrooms
          </p>
        </div>
      )}

      {/* ── Action Row ──────────────────────────────────────────────── */}
      <div className="flex items-center justify-between">
        <Button
          onClick={handleSubmit}
          disabled={!canSubmit}
          className="min-w-[220px]"
        >
          {isSubmitting ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Generating Classrooms...
            </>
          ) : (
            "Save & Generate Classrooms"
          )}
        </Button>
      </div>
    </div>
  );
}
