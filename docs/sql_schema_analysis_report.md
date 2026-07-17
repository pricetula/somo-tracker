# SQL Schema Analysis Report: 000001_initial_schema.up.sql

## Introduction

This report analyzes the SQL migration file `backend/internal/database/migrations/000001_initial_schema.up.sql` to identify areas that could be considered "unimplemented bits" or potential future enhancements based on the schema's design and inline comments.

It's important to note that many `IMPROVE` and `BUG FIX` comments within the migration script itself indicate areas that were identified as incomplete or problematic in a prior state and *have since been addressed* within this very migration. Therefore, this report focuses on areas that are either implicitly suggested for further development or represent common extensions in database design, rather than already resolved issues within this script.

## Key Findings: Potential Areas for Future Enhancement

Based on the detailed review of the provided SQL schema, the following areas could be considered for future implementation or refinement:

### 1. Structured Assessment Type Definitions

*   **Current State:** The `assessment_weight_configs` table uses `assessment_type_code` (a `VARCHAR(50)`) to identify assessment types (e.g., `KNEC_SBA_Project`, `National_KPSEA`).
*   **Observation:** While functional, relying on a string code can lead to inconsistencies and lacks central management for assessment type metadata.
*   **Potential Enhancement:** Implement a dedicated `assessment_types` table. This table could store canonical names, descriptions, and perhaps rules associated with each assessment type. The `assessment_weight_configs` table would then reference this new table via a foreign key, ensuring data integrity and providing a single source of truth for assessment type definitions.

### 2. Comprehensive Audit Trails (`created_by`, `updated_by`)

*   **Current State:** `created_at` and `updated_at` columns are widely implemented across tables, often with triggers. `academic_years` and `academic_terms` also include `version`, `created_by`, and `updated_by` columns.
*   **Observation:** The pattern of `created_by` and `updated_by` is not uniformly applied across all tables that track `created_at` and `updated_at`. For instance, critical tables like `invoices`, `payments`, `medical_incidents`, `cbc_students`, and many others, have `created_at` and `updated_at` but lack `created_by` and `updated_by`.
*   **Potential Enhancement:** Extend the `created_by` and `updated_by` columns (referencing the `users` table) to all significant data tables. This would provide a more complete audit trail, identifying which user initiated or last modified a record, which is crucial for compliance, debugging, and accountability in an educational platform.

### 3. Granular Curriculum Versioning

*   **Current State:** `academic_years` and `academic_terms` include a `version` column. However, curriculum-specific tables like `cbc_learning_areas`, `cbc_strands`, `cbc_sub_strands`, and `performance_indicators` do not have explicit version tracking.
*   **Observation:** While curriculum elements are linked to schools and grade levels, changes to the actual content (names, descriptions, sequence) of strands, sub-strands, or performance indicators over time are not explicitly versioned.
*   **Potential Enhancement:** Introduce versioning mechanisms for curriculum components. This could involve adding `version` columns, `effective_from_date`, and `effective_to_date` to these tables, or implementing a more sophisticated historical tracking system. This would allow for accurate historical reporting of curriculum content and changes mandated by educational authorities.

### 4. Expanded Soft-Delete Strategy

*   **Current State:** The `behavior_categories` table uses an `is_active` boolean column for soft deletion, preserving historical behavior notes.
*   **Observation:** This pattern is effective for maintaining historical data integrity. However, many other core entities (e.g., `cbc_schools`, `cbc_classes`, `users`, `cbc_parents`) do not employ a soft-delete strategy. Deleting these core entities would cascade delete dependent records.
*   **Potential Enhancement:** Evaluate and implement a consistent soft-delete strategy (`is_active` or `deleted_at` columns) for core entities where preserving historical data (even for inactive records) is beneficial for reporting, auditing, or system resilience. This is a design decision that depends on business requirements for data retention.

## Conclusion

The `000001_initial_schema.up.sql` migration provides a robust initial schema, effectively addressing several improvements within its scope. The identified "unimplemented bits" are primarily opportunities for further refinement and expansion, focusing on enhanced auditability, stricter data governance for curriculum content, and a more comprehensive approach to data lifecycle management (e.g., soft deletes). Implementing these enhancements would contribute to a more resilient, auditable, and functionally rich SomoTracker platform.
