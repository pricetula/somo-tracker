# ADR-0021: Backup & Disaster Recovery

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0021 |
| title | Backup & Disaster Recovery |
| status | proposed |
| date | 2026-11-25 |
| author | backend, platform |
| deciders | backend, platform |
| relates_to | ADR-0003, ADR-0007, ADR-0020 |

## Status

proposed

## Context

Somotracker stores irreplaceable school records: assessment results, attendance, student and guardian data, and education configuration. PostgreSQL 18 is the primary database (standard everywhere per AGENTS.md). Redis is used for rate limiting counters and ephemeral OTP/challenge data — Redis data is transient and does not require backup.

The startup integration test (`TestStartup_AppCreatesSuccessfully`) verifies PostgreSQL and Redis are reachable, but there is no documented backup, restore, or disaster recovery strategy. Without one, a database failure or accidental data loss could permanently destroy school records.

## Decision

Adopt a **logical backup + point-in-time recovery (PITR) model** for PostgreSQL:

1. **Physical base backups**: Daily `pg_basebackup` of the PostgreSQL data directory, stored in object storage (S3-compatible) with 35-day retention. Base backups are full copies of the data directory.

2. **Continuous WAL archiving**: PostgreSQL WAL (Write-Ahead Log) segments are archived continuously to object storage via `archive_command`. WAL files enable point-in-time recovery to any second within the retention window.

3. **Logical backups**: Daily `pg_dump` of each organization's schema + data, stored separately from physical backups. Logical backups are used for single-organization restores (e.g. a school accidentally deletes its own data).

4. **Redis**: No backup. Redis is ephemeral — rate limiting counters, OTP challenges, and session lookups can be regenerated or re-authenticated. On Redis failure, the system degrades gracefully (rate limiting falls back to per-process in-memory, sessions remain in PostgreSQL).

5. **Recovery procedures**:
   - **Full cluster restore**: Restore latest base backup + replay WAL to a target time.
   - **Single-organization restore**: Restore logical dump into a new database, then copy organization data back to production.
   - **Cross-region**: Backups replicated to a second region for disaster recovery (RTO < 4 hours, RPO < 1 hour).

6. **Backup verification**: Monthly restore test to a staging environment. Backup integrity verified via `pg_verifybackup` for physical backups and `pg_restore --list` for logical dumps.

7. **Configuration**: Backup schedule, retention, and object storage credentials injected via Doppler at runtime. No backup credentials in repo.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| No backup (data is reproducible) | Simplest, no infrastructure | Assessment results, attendance, guardian consent are not reproducible | Rejected — school records are irreplaceable |
| Physical base backup only | Full cluster restore is fast | No point-in-time recovery; lose up to 24 hours of data | Rejected — schools need granular recovery |
| Logical backup only (`pg_dump`) | Simple, portable | Restore is slow; no PITR; full-cluster restore requires replaying all dumps | Rejected — insufficient for disaster scenarios |
| Logical + physical + WAL archiving (chosen) | PITR, single-org restore, fast full restore | More infrastructure, WAL management complexity | Accepted — correctness and recoverability outweigh complexity |

## Consequences

### Positive

- Point-in-time recovery to any second within the 35-day WAL retention window.
- Single-organization restore via logical dumps without affecting other schools.
- Full cluster disaster recovery with RPO < 1 hour, RTO < 4 hours.
- Redis failure is non-critical (ephemeral data).

### Negative

- WAL archiving adds operational complexity (monitoring archive status, WAL retention).
- Object storage costs for base backups + WAL segments.
- Logical dumps of large organizations may be slow.

### Neutral / Risks

- Backup/restore procedures must be tested regularly (monthly restore test).
- Object storage credentials must be rotated per ADR-0023.
- If WAL archiving fails silently, PITR window shrinks — monitoring alert required.

## Implementation Notes

1. **`archive_command`**: Configure PostgreSQL to copy WAL segments to object storage via `aws s3 cp` or equivalent, with `archive_mode = on`.
2. **`pg_basebackup`**: Daily cron job at off-peak hours, with `--format=tar --compress` for efficient storage.
3. **`pg_dump`**: Per-organization logical dump using `--dbname` filter or schema-level filtering.
4. **Restore tooling**: Scripted `restore_pitr.sh` and `restore_org.sh` runbooks for engineers.
5. **Monitoring**: Alert on backup failure (base backup, WAL archive, logical dump).

## Links

- Session management: `docs/adr/0003-session-management.md`
- Education config migrations: `docs/adr/0020-education-configuration-migrations-seeding.md`
- Secrets rotation: `docs/adr/0023-secrets-rotation.md` (planned)
- Observability: `docs/adr/0022-observability.md` (backup monitoring alerts)
