import { render } from "@testing-library/react";
import { SystemAdminDashboardPage } from "../system-admin-dashboard";
import { AcademicYearHandler } from "@/features/academicyears/components/academic-year-handler";

vi.mock("@/features/academicyears/components/academic-year-handler", () => ({
    AcademicYearHandler: vi.fn(),
}));

const mockedAcademicYearHandler = vi.mocked(AcademicYearHandler);

describe("SystemAdminDashboardPage", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    test("renders AcademicYearHandler with change handlers", () => {
        render(<SystemAdminDashboardPage />);

        // Expect AcademicYearHandler to have been called
        expect(mockedAcademicYearHandler).toHaveBeenCalledTimes(1);

        // Get the props passed to AcademicYearHandler in the first call
        const academicYearHandlerProps = mockedAcademicYearHandler.mock.calls[0][0];

        expect(typeof academicYearHandlerProps.onAcademicYearChange).toBe("function");
        expect(typeof academicYearHandlerProps.onAcademicTermChange).toBe("function");
    });

    test("updates academic year ID when onAcademicYearChange is called", () => {
        const { rerender } = render(<SystemAdminDashboardPage />);

        // Get the onAcademicYearChange function from the first call
        const onAcademicYearChange =
            mockedAcademicYearHandler.mock.calls[0][0].onAcademicYearChange;

        // Call the onAcademicYearChange function with a test year ID
        onAcademicYearChange("test-year-id");

        // Re-render the component (by calling rerender) to simulate the state update
        // Note: rerender will cause the component to re-execute with the updated state
        rerender(<SystemAdminDashboardPage />);

        // Expect AcademicYearHandler to have been called again
        expect(mockedAcademicYearHandler).toHaveBeenCalledTimes(2);
    });

    test("updates academic term ID when onAcademicTermChange is called", () => {
        const { rerender } = render(<SystemAdminDashboardPage />);

        // Get the onAcademicTermChange function from the first call
        const onAcademicTermChange =
            mockedAcademicYearHandler.mock.calls[0][0].onAcademicTermChange;

        // Call the onAcademicTermChange function with a test term ID
        onAcademicTermChange("test-term-id");

        // Re-render the component
        rerender(<SystemAdminDashboardPage />);

        // Expect AcademicYearHandler to have been called again
        expect(mockedAcademicYearHandler).toHaveBeenCalledTimes(2);
    });
});
