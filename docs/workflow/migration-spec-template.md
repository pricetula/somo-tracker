# Migration Spec Template

Use this template to document a new migration before it is created. One spec per table.

## Metadata
* Spec ID:
* Status: draft / approved
* Depends on:
* Touches:

## Table Overview
* Table name:
* Purpose: one sentence

## Columns
| name | type | nullable | default | notes |
|------|------|----------|---------|-------|

## Constraints
* Primary key:
* Unique:
* Check:
* Foreign keys:

## Indexes
| name | columns | partial |
|------|---------|---------|

## RLS
* Enable RLS: yes/no
* Policy name:
* Using clause:

## Soft Delete
* deleted_at present: yes/no
* Partial index needed: yes/no

## Migration Notes
* Extension required:
* Triggers required:
* Backwards compatibility notes:

## Verification
* Tables to assert in integration test:
* Indexes to assert:
* FKs to assert:
