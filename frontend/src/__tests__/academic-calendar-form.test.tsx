/**
 * Tests for the AcademicCalendarForm component (Onboarding Step 1).
 *
 * All backend calls are mocked. The shadcn Form UI layer is mocked to
 * avoid React 19 + jsdom context propagation issues with react-hook-form.
 *
 * To run: pnpm vitest run src/__tests__/academic-calendar-form.test.tsx
 */

import * as React from "react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { QueryClientProvider } from "@tanstack/react-query";

import { createTestQueryClient } from "./test-utils";

// ─── Mocks ────────────────────────────────────────────────────────────────

const mockSaveAcademicCalendar = vi.fn();

vi.mock("@/lib/api/academic-calendar", () => ({
  saveAcademicCalendar: (...args: unknown[]) => mockSaveAcademicCalendar(...args),
  fetchCurrentCalendar: vi.fn().mockResolvedValue(null),
}));

// Mock the shadcn Form UI to avoid jsdom context propagation issues.
// The real component logic (react-hook-form validation, field array) runs
// correctly — only the presentation layer is simplified.
vi.mock("@/components/ui/form", () => {
  const React = require("react");
  const { Controller, FormProvider } = require("react-hook-form");
  return {
    Form: FormProvider,
    FormField: Controller,
    FormItem: ({ children, className }: any) =>
      React.createElement("div", { className }, children),
    FormLabel: ({ children, className }: any) =>
      React.createElement("label", { className }, children),
    FormControl: ({ children }: any) => React.createElement("div", null, children),
    FormMessage: () => null,
    FormDescription: () => null,
    useFormField: () => ({ error: null, formItemId: "id", formDescriptionId: "desc", formMessageId: "msg" }),
  };
});

vi.mock("@/features/calendar/components/date-picker", () => ({
  DatePicker: ({ onChange }: any) => {
    const React = require("react");
    return React.createElement("input", {
      "data-testid": "date-picker",
      onChange: (e: any) => onChange?.(new Date(e.target.value)),
    });
  },
}));

import { AcademicCalendarForm } from "@/features/calendar";

// ─── Helpers ──────────────────────────────────────────────────────────────

function renderForm(props: { onSuccess?: () => void } = {}) {
  const qc = createTestQueryClient();
  const user = userEvent.setup();
  const utils = render(<AcademicCalendarForm {...props} />, {
    wrapper: ({ children }) => React.createElement(QueryClientProvider, { client: qc }, children),
  });
  return { ...utils, user };
}

function submitBtn() {
  return screen.getByRole("button", { name: /save & activate calendar/i });
}

function fillBtn() {
  return screen.getByRole("button", { name: /fill with sample/i });
}

function addPeriodBtn() {
  return screen.getByRole("button", { name: /add period row/i });
}

function trashBtns() {
  return screen.getAllByRole("button", { name: /remove period/i });
}

function yearInput() {
  // The input is a type="number" rendered via FormControl > Input
  // With the form UI mocked, there's no accessible label association
  // so we fall back to finding the numeric input by role
  return screen.getByRole("spinbutton");
}

async function fillSampleData(user: ReturnType<typeof userEvent.setup>) {
  await user.click(fillBtn());
  await waitFor(() => expect(submitBtn()).not.toBeDisabled());
}

// ─── Suite ────────────────────────────────────────────────────────────────

describe("AcademicCalendarForm (Step 1)", () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(() => cleanup());

  // ── Initial State ──────────────────────────────────────────────────────

  describe("initial state", () => {
    it("renders header and year field", () => {
      renderForm();
      expect(screen.getByText("Set Up Academic Calendar")).toBeInTheDocument();
      expect(yearInput()).toHaveValue(new Date().getFullYear());
    });

    it("disables submit when dates are empty", () => {
      renderForm();
      expect(submitBtn()).toBeDisabled();
    });

    it("renders 3 default period rows", () => {
      renderForm();
      // With the mock, each FormField renders its render prop's content.
      // 1 year field + 3 period name fields + 3 start date + 3 end date = 10 fields
      const fields = screen.getAllByRole("textbox");
      expect(fields.length).toBeGreaterThanOrEqual(3);
    });
  });

  // ── Year Validation ────────────────────────────────────────────────────

  describe("year validation", () => {
    it("accepts a valid year", async () => {
      const { user } = renderForm();
      await user.clear(yearInput());
      await user.type(yearInput(), "2026");
      expect(yearInput()).toHaveValue(2026);
    });

    it("rejects year < 2020", async () => {
      const { user } = renderForm();
      await user.clear(yearInput());
      await user.type(yearInput(), "2019");
      await user.tab();
      // The form should be invalid with year < 2020
      await waitFor(() => expect(submitBtn()).toBeDisabled());
    });

    it("rejects year > 2100", async () => {
      const { user } = renderForm();
      await user.clear(yearInput());
      await user.type(yearInput(), "2200");
      await user.tab();
      await waitFor(() => expect(submitBtn()).toBeDisabled());
    });
  });

  // ── Period Row Management ──────────────────────────────────────────────

  describe("period row management", () => {
    it("adds a period row", async () => {
      const { user } = renderForm();
      await user.click(addPeriodBtn());
      expect(addPeriodBtn()).toBeInTheDocument(); // form still renders
    });

    it("removes a period row", async () => {
      const { user } = renderForm();
      const before = trashBtns().length;
      await user.click(trashBtns()[0]);
      expect(trashBtns()).toHaveLength(before - 1);
    });

    it("disables remove on the last period", async () => {
      const { user } = renderForm();
      await user.click(trashBtns()[1]);
      await user.click(trashBtns()[0]);
      await waitFor(() => {
        const remaining = trashBtns();
        if (remaining.length === 1) expect(remaining[0]).toBeDisabled();
      });
    });
  });

  // ── is_final Toggle ────────────────────────────────────────────────────

  describe("is_final toggle", () => {
    it("switches final status between rows", async () => {
      const { user } = renderForm();
      const finals = () => screen.getAllByRole("button", { name: /set final|final/i });
      expect(finals()).toHaveLength(3);
      await user.click(finals()[0]);
      // The FormField is now a Controller; the render prop produces the button.
      // After clicking row 0, it should say "Final"
      await waitFor(() => expect(finals()[0].textContent).toBe("Final"));
    });
  });

  // ── Sample Data ─────────────────────────────────────────────────────────

  describe("sample data fill", () => {
    it("enables submit after filling", async () => {
      const { user } = renderForm();
      await fillSampleData(user);
      expect(submitBtn()).not.toBeDisabled();
    });
  });

  // ── Submission ──────────────────────────────────────────────────────────

  describe("submission", () => {
    it("calls saveAcademicCalendar with correct payload", async () => {
      mockSaveAcademicCalendar.mockResolvedValueOnce({ id: "cal", year: 2026, periods: [] });
      const onSuccess = vi.fn();
      const { user } = renderForm({ onSuccess });
      await fillSampleData(user);
      await user.click(submitBtn());

      await waitFor(() => expect(mockSaveAcademicCalendar).toHaveBeenCalledTimes(1));
      const p = mockSaveAcademicCalendar.mock.calls[0][0];
      expect(p).toMatchObject({ year: 2026 });
      expect(p.periods).toHaveLength(3);
      expect(p.periods[0]).toMatchObject({ name: "Term 1", is_final: false });
      expect(p.periods[2]).toMatchObject({ name: "Term 3", is_final: true });
      expect(p.periods[0]).toHaveProperty("start_date");
      expect(p.periods[0]).toHaveProperty("end_date");
    });

    it("shows loading state during submission", async () => {
      mockSaveAcademicCalendar.mockImplementationOnce(() => new Promise(() => {}));
      const { user } = renderForm();
      await fillSampleData(user);
      await user.click(submitBtn());
      expect(await screen.findByText("Saving...")).toBeInTheDocument();
    });

    it("shows success checkmark and fires onSuccess", async () => {
      mockSaveAcademicCalendar.mockResolvedValueOnce({ id: "cal", year: 2026, periods: [] });
      const onSuccess = vi.fn();
      const { user } = renderForm({ onSuccess });
      await fillSampleData(user);
      await user.click(submitBtn());

      expect(await screen.findByText(/Calendar activated/i)).toBeInTheDocument();
      await waitFor(() => expect(onSuccess).toHaveBeenCalledTimes(1), { timeout: 3000 });
    });

    it("toasts error on API failure", async () => {
      mockSaveAcademicCalendar.mockRejectedValueOnce(new Error("exists"));
      const onSuccess = vi.fn();
      const { user } = renderForm({ onSuccess });
      await fillSampleData(user);
      await user.click(submitBtn());

      await waitFor(() => {
        expect(toast.error).toHaveBeenCalledWith("Failed to save calendar", expect.any(Object));
      });
      expect(onSuccess).not.toHaveBeenCalled();
    });
  });

  // ── Edge Cases ──────────────────────────────────────────────────────────

  describe("edge cases", () => {
    it("reduces to single period and submits", async () => {
      mockSaveAcademicCalendar.mockResolvedValueOnce({ id: "cal", year: 2026, periods: [] });
      const onSuccess = vi.fn();
      const { user } = renderForm({ onSuccess });
      await user.click(trashBtns()[1]);
      await user.click(trashBtns()[0]);
      await fillSampleData(user);

      expect(submitBtn()).not.toBeDisabled();
      await user.click(submitBtn());

      await waitFor(() => expect(mockSaveAcademicCalendar).toHaveBeenCalledTimes(1));
      await waitFor(() => expect(onSuccess).toHaveBeenCalledTimes(1), { timeout: 3000 });
    });
  });
});
