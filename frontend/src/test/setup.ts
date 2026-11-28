import '@testing-library/jest-dom/vitest'

// Mock IntersectionObserver for jsdom (used by DataTable infinite scroll)
;(globalThis as any).IntersectionObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() { return [] }
  root = null
  rootMargin = ''
  thresholds = []
}
