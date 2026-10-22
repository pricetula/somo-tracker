# Feature Specification Template

> Copy this file to `docs/features/FEATURE-<slug>.md` and fill it out.

# Feature: <Feature Name>

**Status:** Draft | Proposed | Accepted | In Progress | Done | Deprecated
**Owner:** 
**Created:** YYYY-MM-DD
**Last Updated:** YYYY-MM-DD
**Related Issues:** #
**Affected Layers:** backend | frontend | public | docs | infra

## Summary
One sentence description of what this feature does and why it matters.

## Problem Statement
Who experiences the pain? What is broken or missing today?
- Context
- User pain points
- Business impact

## Solution Overview
High level description of the proposed solution.

### User Stories
- As a <role>, I want <capability> so that <outcome>
- ...

### Acceptance Criteria
- [ ] ...
- [ ] ...

## Scope

### Backend
- API changes: 
- DB changes / migrations:
- Error contract compliance: see `backend/internal/api/errors.go`
- RLS / multi-tenancy considerations:

### Frontend
- Pages / flows:
- Components / shadcn usage:
- No dumb back buttons per AGENTS.md

### Public / Marketing
- Changes needed:

### Non-Goals
Explicitly list what is out of scope.

## Design Notes
Relevant ADRs, diagrams, links.

## Open Gaps / Risks
- ...

## Test Plan
- Unit tests
- Integration tests
- Playwright / E2E scenarios (Ignore)

## Rollout
- Feature flag? 
- Migration steps
- Monitoring

## References
- Links to RFC / ADR / Issues
