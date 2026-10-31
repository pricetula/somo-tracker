---
id: PRODUCT-BRIEF
title: Product Brief
status: draft
depends_on: []
touches: []
---

# Product Brief

## One-liner
Give Kenyan schools PP1 to Grade 9 under CBC one fast, reliable system for attendance, assessment, welfare, and fees.

## Target users
- Kenyan primary and junior secondary schools, day and boarding, PP1 to Grade 9 under CBC
- Platform Operator / Super Admin
- Tenant Owner
- Headteacher
- School Administrator / Registrar
- Teacher
- Finance Officer
- Guardian
- Student via short-lived quiz token, no account

## Core value proposition
Mobile-first, intermittently-connected platform that consolidates attendance, CBC assessment and report cards, welfare notes, and fees into one tenant-scoped system. Reduces admin time on attendance reporting, report card preparation and fee statements while giving guardians timely visibility.

### Assumptions
- A1: Most teachers and guardians use mobile phones with intermittent connectivity
- A2: Email is the only notification channel in MVP; SMS is Phase 2
- A3: English is launch language; Swahili later

## Success metrics
- SM-1: Median time to complete daily class roll-call under 60 seconds on a mid-range phone
- SM-2: Admin time on attendance reporting, report card preparation, and fee statements reduced by target percentage vs pilot baseline at onboarding and end of Term 1
- SM-3: 500-student CSV import completes in under 30 seconds with per-row error reporting
- SM-4: At least 70% of active guardians view at least one update per term

## Constraints and non-goals

### Constraints
- Hosted in Frankfurt, EU
- Mobile-first responsive design; core tasks completable after 10-minute walkthrough
- Kenya Data Protection Act 2019 compliance: lawful basis, guardian consent for child data, ODPC registration as required, data subject access/correction, breach notification, retention schedule
- Encryption in transit TLS 1.2+ and at rest; secrets and PII never logged
- Role checks enforced server-side on every request

### In scope
PP1, PP2, Grades 1-6 Primary, Grades 7-9 Junior Secondary, day and boarding schools

### Out of scope for this version
Senior School Grade 10+, payroll, library, inventory, M-Pesa gateway integration, NEMIS sync

### Non-goals
Per-period attendance, per-school report card customization, billing in MVP, SMS notifications in MVP, Swahili launch, statistical at-risk prediction
