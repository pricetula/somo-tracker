# ADR-2026-10-15: Stats Architecture — Summary Tables with Background Workers

**Status:** Proposed
**Date:** 2026-10-15
**Deciders:** Platform Team

## Context
Multiple features require summary statistics: attendance %, guardian links, teacher workload, class enrollments, student KPIs. Initial specs assumed on-the-fly calculations or materialized views.

Product requirement: no on-the-fly aggregations, no materialized views. Prefer summary/stat tables updated asynchronously when resources change.

## Decision
Use summary tables updated by background workers via domain events. Reads come from summary tables only.

## Consequences
**Positive**
* Fast reads for list pages and dashboards
* Predictable DB load
* Decouples read performance from write volume

**Negative**
* Eventual consistency; stats may lag writes by seconds
* Requires event emission and worker reliability

## Alternatives Considered
* On-the-fly aggregation: rejected due to performance at scale
* Materialized views: rejected per product preference

## Implementation Notes
* Reuse Asynq for background tasks
* Emit events from services on create/update/delete
* Workers upsert summary tables
* Add reconciliation job nightly

## References
* `docs/planning/stats-summary-tables-plan.md`
