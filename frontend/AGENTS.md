# AI Agent Instructions: Next.js Feature-Module Architecture

A Next.js (App Router), TypeScript, and Domain-Driven/Feature-Module design. Maintain a highly scalable, cohesive, and loosely coupled codebase.

---

## 1. Core Architecture Style

The project is organized by **Feature Modules** inside a top-level directory. `src/app` should only include routing files, no custom components.

Each feature should be self-contained, encapsulating its own logic, UI, and state.

### Directory Structure Blueprint

When creating or modifying a feature, adhere strictly to this isolated module structure:

```text
src/
├── app/                  # Next.js Routing Layer (Keep clean & lean)
│   ├── layout.tsx
│   ├── page.tsx          # Imports and renders Feature Containers
│   └── dashboard/
│       └── page.tsx      # Imports <DashboardContainer />
│
├── features/             # Feature-Module Layer (The Core Business Logic)
│   ├── auth/             # Example Feature: Authentication
│   └── analytics/        # Example Feature: Analytics
│       ├── components/   # Feature-specific presentational UI
│       │   ├── analytics-chart.tsx
│       │   └── analytics-summary.tsx
│       ├── hooks/        # Feature-specific hooks (data fetching/state)
│       │   └── use-analytics-data.ts
│       ├── services/     # API clients, server actions, or SDK wrappers
│       │   └── analytics-api.ts
│       ├── types/        # TypeScript interfaces unique to this feature
│       │   └── index.ts
│       └── index.ts      # PUBLIC API: Strict entry point for the feature
│
├── components/           # Truly GLOBAL, generic UI components only (e.g., Shadcn/Button)
└── lib/                  # Truly GLOBAL utilities (e.g., prisma, tailwind-merge)
```
