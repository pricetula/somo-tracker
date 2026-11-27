import '@testing-library/jest-dom/vitest'
import { vi } from 'vitest'

// Mock IntersectionObserver for jsdom (used by DataTable infinite scroll)
const mockIntersectionObserver = {
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}

;(globalThis as any).IntersectionObserver = vi.fn(() => mockIntersectionObserver)
