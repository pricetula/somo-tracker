/**
 * Tests for the ClassStreamGenerator component (Onboarding Step 2).
 *
 * Coverage scope:
 *   - Tag input: add single tag, remove tag, duplicate prevention, paste
 *   - Live preview grid: cross-multiplication with CBE_GRADE_TIERS
 *   - Empty state when no streams
 *   - Button guard: disabled with empty streams, enabled with streams
 *   - Submission: mock API call, loading lock, success checkmark, onSuccess
 *   - Error handling on API failure
 *   - Edge cases: zero streams, single stream, many streams
 *
 * To run: pnpm vitest run src/__tests__/class-stream-generator.test.tsx
 */

import * as React from "react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";

import { QueryClientProvider } from "@tanstack/react-query";
import { ClassStreamGenerator } from "@/features/classes";
import { CBE_GRADE_TIERS } from "@/features/classes/types";
import { createTestQueryClient } from "./test-utils";

// ─── Mock the API module ─────────────────────────────────────────────────

const mockGenerateClasses = vi.fn();

vi.mock("@/lib/api/classes", () => ({
  generateClasses: (...args: unknown[]) => mockGenerateClasses(...args),
  fetchClasses: vi.fn().mockResolvedValue([]),
}));

// ─── Shared state ─────────────────────────────────────────────────────────

let queryClient: ReturnType<typeof createTestQueryClient>;

// ─── Helpers ──────────────────────────────────────────────────────────────

/** Render the generator with a fresh QueryClient. */
function renderGenerator(props: { onSuccess?: () => void } = {}) {
  queryClient = createTestQueryClient();
  return render(
    <ClassStreamGenerator {...props} />,
    {
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      ),
    },
  );
}

/** Type a stream name into the tag input and press Enter. */
async function addStream(user: ReturnType<typeof userEvent.setup>, name: string) {
  const input = screen.getByRole("textbox", { name: /stream name input/i });
  await user.type(input, `${name}{Enter}`);
}

/** Get all stream tag chip texts from the visible chips area. */
function getStreamChips(): string[] {
  const chips = screen.queryAllByRole("button", { name: /remove/i });
  // The aria-label contains the stream name + "Remove" — extract it
  return chips.map((btn) => btn.getAttribute("aria-label")?.replace(/remove /i, "").trim() ?? "");
}

/** Get all preview grid cell labels. */
function getGridCells(): string[] {
  const cells = screen.queryAllByTestId("preview-cell");
  return cells.map((el) => el.textContent?.trim() ?? "");
}

/** Get the "Save & Generate" button. */
function getSubmitBtn() {
  return screen.getByRole("button", { name: /save & generate/i });
}

// ─── Suite ────────────────────────────────────────────────────────────────

describe("ClassStreamGenerator (Step 2)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGenerateClasses.mockResolvedValue({
      classes: [],
      total_created: 0,
      streams: [],
      grade_names: CBE_GRADE_TIERS,
    });
  });

  afterEach(() => {
    cleanup();
  });

  // ── Initial Render ──────────────────────────────────────────────────

  describe("initial render", () => {
    it("renders the step title and header", () => {
      renderGenerator();
      expect(screen.getByText(/step 2: establish your classes/i)).toBeInTheDocument();
      expect(screen.getByText(/define your streams/i)).toBeInTheDocument();
    });

    it("shows the initial placeholder in the tag input", () => {
      renderGenerator();
      const input = screen.getByRole("textbox", { name: /stream name input/i });
      expect(input).toHaveAttribute(
        "placeholder",
        expect.stringMatching(/type a stream name/i),
      );
    });

    it("shows the dashed empty state when no streams exist", () => {
      renderGenerator();
      expect(screen.getByText(/add a stream above/i)).toBeInTheDocument();
    });

    it("disables the submit button when no streams", () => {
      renderGenerator();
      expect(getSubmitBtn()).toBeDisabled();
    });
  });

  // ── Tag Input Mechanics ─────────────────────────────────────────────

  describe("tag input mechanics", () => {
    it("adds a stream tag when typing and pressing Enter", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      expect(getStreamChips()).toContain("East");
    });

    it("adds multiple stream tags", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      await addStream(user, "West");
      await addStream(user, "North");
      expect(getStreamChips()).toEqual(["East", "West", "North"]);
    });

    it("clears the input field after adding a tag", async () => {
      const user = userEvent.setup();
      renderGenerator();
      const input = screen.getByRole("textbox", { name: /stream name input/i });
      await user.type(input, "East{Enter}");
      expect(input).toHaveValue("");
    });

    it("removes a tag when clicking the × button", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      await addStream(user, "West");

      // Click the first × button
      const removeBtns = screen.getAllByRole("button", { name: /remove/i });
      await user.click(removeBtns[0]);

      expect(getStreamChips()).toEqual(["West"]);
    });

    it("does not add duplicate tags (case-insensitive)", async () => {
      const user = userEvent.setup({ delay: 10 });
      renderGenerator();
      await addStream(user, "East");
      await waitFor(() => expect(getStreamChips()).toEqual(["East"]));

      // Typing "east" then Enter — the dedup check should reject it
      await addStream(user, "east");

      // Should still only have the original chip
      await waitFor(() => expect(getStreamChips()).toHaveLength(1));
      expect(getStreamChips()[0]).toBe("East");
    });

    it("trims whitespace from stream names", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "  East  ");
      expect(getStreamChips()).toContain("East");
    });

    it("rejects empty input", async () => {
      const user = userEvent.setup();
      renderGenerator();
      const input = screen.getByRole("textbox", { name: /stream name input/i });
      await user.type(input, "   {Enter}");
      expect(getStreamChips()).toHaveLength(0);
    });

    it("changes placeholder after first tag is added", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      const input = screen.getByRole("textbox", { name: /stream name input/i });
      expect(input).toHaveAttribute("placeholder", expect.stringMatching(/add another/i));
    });

    it("removes the last tag on Backspace when input is empty", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      await addStream(user, "West");

      const input = screen.getByRole("textbox", { name: /stream name input/i });
      await user.type(input, "{Backspace}");

      expect(getStreamChips()).toEqual(["East"]);
    });
  });

  // ── Paste Support ──────────────────────────────────────────────────

  describe("paste support", () => {
    it("adds comma-separated streams on paste", async () => {
      const user = userEvent.setup();
      renderGenerator();
      const input = screen.getByRole("textbox", { name: /stream name input/i });
      await user.click(input);
      await user.paste("East,West,North");
      expect(getStreamChips()).toEqual(["East", "West", "North"]);
    });

    it("adds newline-separated streams on paste", async () => {
      const user = userEvent.setup();
      renderGenerator();
      const input = screen.getByRole("textbox", { name: /stream name input/i });
      await user.click(input);
      await user.paste("East\nWest\nNorth");
      expect(getStreamChips()).toEqual(["East", "West", "North"]);
    });

    it("deduplicates pasted streams against existing tags", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");

      const input = screen.getByRole("textbox", { name: /stream name input/i });
      await user.click(input);
      await user.paste("East,West,EAST");

      expect(getStreamChips()).toEqual(["East", "West"]);
    });
  });

  // ── Live Preview Grid ──────────────────────────────────────────────

  describe("live preview grid", () => {
    it("shows preview cells when streams are added", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");

      const cells = getGridCells();
      expect(cells).toHaveLength(CBE_GRADE_TIERS.length);
      expect(cells[0]).toBe("Grade 1 East");
      expect(cells[cells.length - 1]).toBe("Grade 4 East");
    });

    it("updates grid when a tag is removed", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      await addStream(user, "West");

      expect(getGridCells()).toHaveLength(CBE_GRADE_TIERS.length * 2);

      // Remove "West" (second chip)
      const removeBtns = screen.getAllByRole("button", { name: /remove/i });
      await user.click(removeBtns[1]);

      expect(getGridCells()).toHaveLength(CBE_GRADE_TIERS.length);
    });

    it("hides preview grid when all streams are removed", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      expect(getGridCells()).toHaveLength(CBE_GRADE_TIERS.length);

      await user.click(screen.getByRole("button", { name: /remove/i }));
      expect(screen.getByText(/add a stream above/i)).toBeInTheDocument();
    });

    it("generates the correct cross product order", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "A");
      await addStream(user, "B");

      const cells = getGridCells();
      expect(cells[0]).toBe("Grade 1 A");
      expect(cells[1]).toBe("Grade 1 B");
      expect(cells[2]).toBe("Grade 2 A");
      expect(cells[3]).toBe("Grade 2 B");
    });
  });

  // ── Button Guard ───────────────────────────────────────────────────

  describe("button guard", () => {
    it("is disabled when no streams are added", () => {
      renderGenerator();
      expect(getSubmitBtn()).toBeDisabled();
    });

    it("becomes enabled when at least one stream is added", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      expect(getSubmitBtn()).not.toBeDisabled();
    });

    it("becomes disabled again when all streams are removed", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      expect(getSubmitBtn()).not.toBeDisabled();

      await user.click(screen.getByRole("button", { name: /remove/i }));
      expect(getSubmitBtn()).toBeDisabled();
    });
  });

  // ── Form Submission ────────────────────────────────────────────────

  describe("form submission", () => {
    it("calls generateClasses with the correct payload", async () => {
      mockGenerateClasses.mockResolvedValueOnce({
        classes: [], total_created: 8, streams: ["East", "West"], grade_names: CBE_GRADE_TIERS,
      });
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      await addStream(user, "West");
      await user.click(getSubmitBtn());

      await waitFor(() => {
        expect(mockGenerateClasses).toHaveBeenCalledTimes(1);
        expect(mockGenerateClasses).toHaveBeenCalledWith({ streams: ["East", "West"] });
      });
    });

    it("shows loading text during submission", async () => {
      mockGenerateClasses.mockImplementationOnce(() => new Promise(() => {}));
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "East");
      await user.click(getSubmitBtn());
      expect(await screen.findByText(/generating classrooms/i)).toBeInTheDocument();
    });

    it("shows success checkmark and calls onSuccess", async () => {
      mockGenerateClasses.mockResolvedValueOnce({
        classes: [], total_created: 4, streams: ["East"], grade_names: CBE_GRADE_TIERS,
      });
      const onSuccess = vi.fn();
      const user = userEvent.setup();
      renderGenerator({ onSuccess });
      await addStream(user, "East");
      await user.click(getSubmitBtn());

      expect(await screen.findByText(/classrooms generated/i)).toBeInTheDocument();
      await waitFor(() => expect(onSuccess).toHaveBeenCalledTimes(1), { timeout: 3000 });
    });

    it("shows error toast on API failure", async () => {
      mockGenerateClasses.mockRejectedValueOnce(new Error("Stream already exists"));
      const onSuccess = vi.fn();
      const user = userEvent.setup();
      renderGenerator({ onSuccess });
      await addStream(user, "East");
      await user.click(getSubmitBtn());

      await waitFor(() => {
        expect(toast.error).toHaveBeenCalledWith(
          "Failed to generate classrooms",
          expect.any(Object),
        );
      });
      expect(onSuccess).not.toHaveBeenCalled();
    });
  });

  // ── Edge Cases ────────────────────────────────────────────────────

  describe("edge cases", () => {
    it("handles a single stream gracefully", async () => {
      mockGenerateClasses.mockResolvedValueOnce({
        classes: [], total_created: 4, streams: ["Default"], grade_names: CBE_GRADE_TIERS,
      });
      const onSuccess = vi.fn();
      const user = userEvent.setup();
      renderGenerator({ onSuccess });
      await addStream(user, "Default");

      expect(getGridCells()).toHaveLength(CBE_GRADE_TIERS.length);
      expect(getGridCells()[0]).toBe("Grade 1 Default");

      await user.click(getSubmitBtn());
      expect(await screen.findByText(/classrooms generated/i)).toBeInTheDocument();
      await waitFor(() => expect(onSuccess).toHaveBeenCalledTimes(1), { timeout: 3000 });
    });

    it("handles many streams correctly", async () => {
      const user = userEvent.setup();
      renderGenerator();
      const streams = Array.from({ length: 10 }, (_, i) => `Stream-${i + 1}`);
      for (const s of streams) {
        await addStream(user, s);
      }

      expect(getGridCells()).toHaveLength(CBE_GRADE_TIERS.length * 10);
      // The preview count badge should reflect the total
      const badges = screen.getAllByText(/total/i);
      expect(badges.some((b) => b.textContent === "40 total")).toBe(true);
    });

    it("allows special characters in stream names", async () => {
      const user = userEvent.setup();
      renderGenerator();
      await addStream(user, "Section-A/B");
      expect(getStreamChips()).toContain("Section-A/B");
      expect(screen.getByText("Grade 1 Section-A/B")).toBeInTheDocument();
    });
  });
});
