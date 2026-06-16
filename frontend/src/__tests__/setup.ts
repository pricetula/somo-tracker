import "@testing-library/jest-dom/vitest";
import { vi } from "vitest";

// ─── Global Mocks ─────────────────────────────────────────────────────────

/**
 * Mock sonner toast to prevent rendering toast containers in test DOM.
 * Individual tests can still assert on calls via `vi.mocked(toast)`.
 */
vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warning: vi.fn(),
    loading: vi.fn(),
    dismiss: vi.fn(),
  },
  Toaster: vi.fn(() => null),
}));
