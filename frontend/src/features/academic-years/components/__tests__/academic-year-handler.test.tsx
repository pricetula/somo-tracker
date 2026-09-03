import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AcademicYearHandler } from "../academic-year-handler";

// Mock the useAcademicYears hook
vi.mock("@/features/academic-years/hooks", () => ({
    useAcademicYears: () => ({
        data: [
            {
                id: "year1",
                name: "Academic Year 1",
                is_current: true,
                start_date: "2023-08-01",
                end_date: "2024-05-31",
                terms: [
                    {
                        id: "term1",
                        name: "Term 1",
                        term_number: "1",
                        start_date: "2023-08-01",
                        end_date: "2023-12-15",
                        is_current: true,
                        is_final: false,
                    },
                    {
                        id: "term2",
                        name: "Term 2",
                        term_number: "2",
                        start_date: "2024-01-15",
                        end_date: "2024-05-31",
                        is_current: false,
                        is_final: true,
                    },
                ],
            },
            {
                id: "year2",
                name: "Academic Year 2",
                is_current: false,
                start_date: "2024-08-01",
                end_date: "2025-05-31",
                terms: [
                    {
                        id: "term3",
                        name: "Term 1",
                        term_number: "1",
                        start_date: "2024-08-01",
                        end_date: "2024-12-15",
                        is_current: true,
                        is_final: false,
                    },
                    {
                        id: "term4",
                        name: "Term 2",
                        term_number: "2",
                        start_date: "2025-01-15",
                        end_date: "2025-05-31",
                        is_current: false,
                        is_final: true,
                    },
                ],
            },
        ],
    }),
}));

describe("AcademicYearHandler", () => {
    test("sets initial academic year and term to current ones", async () => {
        const onAcademicYearChange = vi.fn();
        const onAcademicTermChange = vi.fn();

        render(
            <TooltipProvider>
                <AcademicYearHandler
                    onAcademicYearChange={onAcademicYearChange}
                    onAcademicTermChange={onAcademicTermChange}
                />
            </TooltipProvider>
        );

        // Wait for the initial load to run (which sets the initial year and term)
        await Promise.resolve();

        expect(onAcademicYearChange).toHaveBeenCalledWith("year1");
        expect(onAcademicTermChange).toHaveBeenCalledWith("term1");
    });

    test("changes academic year and updates term to current term of new year", async () => {
        const onAcademicYearChange = vi.fn();
        const onAcademicTermChange = vi.fn();

        render(
            <TooltipProvider>
                <AcademicYearHandler
                    onAcademicYearChange={onAcademicYearChange}
                    onAcademicTermChange={onAcademicTermChange}
                />
            </TooltipProvider>
        );

        // Wait for initial load
        await Promise.resolve();

        // Find the academic year combobox and click it
        const yearCombobox = screen.getByPlaceholderText(/select a academic year/i);
        await userEvent.click(yearCombobox);

        // Select the second academic year (Academic Year 2)
        const year2Option = screen.getByRole("option", { name: /academic year 2/i });
        await userEvent.click(year2Option);

        // Expect the year change callback to be called with year2
        expect(onAcademicYearChange).toHaveBeenCalledWith("year2");
        // Expect the term change callback to be called with the current term of year2, which is term3
        expect(onAcademicTermChange).toHaveBeenCalledWith("term3");
    });

    test("changes term when term combobox is used", async () => {
        const onAcademicYearChange = vi.fn();
        const onAcademicTermChange = vi.fn();

        render(
            <TooltipProvider>
                <AcademicYearHandler
                    onAcademicYearChange={onAcademicYearChange}
                    onAcademicTermChange={onAcademicTermChange}
                />
            </TooltipProvider>
        );

        // Wait for initial load (which sets year1 and term1)
        await Promise.resolve();

        // Clear the mock call history to isolate the term change
        onAcademicYearChange.mockClear();
        onAcademicTermChange.mockClear();

        // Find the term combobox and click it
        const termCombobox = screen.getByPlaceholderText(/select a academic term/i);
        await userEvent.click(termCombobox);

        // Select the second term (Term 2)
        const term2Option = screen.getByRole("option", { name: /term 2/i });
        await userEvent.click(term2Option);

        // Expect the term change callback to be called with term2
        expect(onAcademicTermChange).toHaveBeenCalledWith("term2");
        // Year should not change
        expect(onAcademicYearChange).not.toHaveBeenCalled();
    });
});
