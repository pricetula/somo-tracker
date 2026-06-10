# Somotracker Monorepo

An intelligent, multi-tenant educational analytics platform designed for real-time academic performance tracking, historical subject trends, and early warning interventions.

## 🚀 Repository Structure

This project is organized as a monorepo splitting the public marketing presence, core application frontend, high-performance backend API, and system documentation.

```text
.
├── .github/          # CI/CD workflows and automation pipelines
├── ./backend/        # Go (Fiber) REST API — Core business logic & analytics engine
├── ./docs/           # Standardized Markdown templates and system feature specs
├── ./frontend/       # Next.js (App Router) — Multi-tenant educational dashboard
└── ./public/         # Svelte — High-conversion, lightweight marketing website
```

---
## 📦 Sub-Project Overviews

### 1. `./public` (Marketing Site)
* **Technology:** Svelte
* **Scope:** Public-facing landing pages, product value propositions, and conversion funnels.
* **Focus:** Speed, SEO optimization, and messaging centered strictly on **High-ROI & No Time Waste** for educational institutions.

### 2. `./frontend` (Core Application)
* **Technology:** Next.js (Latest App Router), TanStack Query
* **Scope:** Role-based dashboards serving School Admins, Teachers, and Support Staff.
* **Data Fetching Strategy:** Pure REST pipelines via TanStack Query. **No Polling, No Server-Sent Events (SSE)**. State synchronization is driven by standard data mutations and explicit invalidations (e.g., hard refreshes or cache-busting on action).

### 3. `./backend` (API Engine)
* **Technology:** Go (Fiber framework), PostgreSQL (Neon DB), Asynq
* **Scope:** High-performance REST API handling multi-tenant data isolation, analytics calculations, and asynchronous background worker processing (via Asynq tasks).

### 4. `./docs` (System Documentation)
* **Scope:** Source of truth for features, business rules, and technical specifications.
* **Rule for Agents:** Feature files use standardized Markdown templates containing strict implementation checklists. Code examples are omitted from tasks to allow agents to derive implementation patterns natively from the existing codebase.

---

## 🏛️ Core Architectural Principles

* **Multi-Tenant Isolation:** Enforced strictly via PostgreSQL Row-Level Security (RLS) at the database layer. All queries must respect tenant boundaries.
* **Asynchronous Operations:** Heavy analytical tasks, threshold evaluations (e.g., low-attendance tracking), and notifications are offloaded to background workers using `Asynq`.
* **State & Communication:** Keep the boundary between frontend and backend predictable. UI components rely on solid HTTP verbs; real-time streaming architectures are intentionally excluded to keep the stack lean and maintainable.