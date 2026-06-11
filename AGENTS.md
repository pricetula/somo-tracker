Somotracker Application
This document establishes the directories available for our application.

📂 1. Monorepo Structural Blueprint
The Somotracker platform is organized as a single, unified monorepo. Agents must strictly isolate code changes to their respective top-level directories:

├── .github/          # CI/CD workflows and automation pipelines
├── ./backend/        # Go (Fiber) REST API — Core business logic & analytics engine
├── ./docs/           # Standardized Markdown templates and system feature specs
├── ./frontend/       # Next.js (App Router) — Multi-tenant educational dashboard
└── ./public/         # Svelte — High-conversion, lightweight marketing website