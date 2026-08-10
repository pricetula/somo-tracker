# Assessments Endpoints

This document outlines the API endpoints related to assessment management, including grading scale profiles, assessment sessions, student scores and grades, parent views, and various summary and projection endpoints.

## Grading Scale Profiles

### 1. Create Grading Scale Profile

*   **URL:** `/api/v1/grading/profiles`
*   **Method:** `POST`
*   **Description:** Creates a new grading scale profile with its percentage-to-level ranges. Requires at least EE, ME, and AE ranges. The profile name is immutable after creation.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Request Body (CreateScaleProfilePayload):
    *   `name` (string, required): The name of the grading scale profile (e.g., "Grade 4 Standard Conversion").
    *   `ranges` (array of objects, required):
        *   `performance_level` (string, required): The CBC performance level (e.g., "EE", "ME", "AE", "BE").
        *   `min_percentage` (integer, required): The minimum percentage for this level.
        *   `max_percentage` (integer, required): The maximum percentage for this level.
        *   `default_percentage_mapping` (integer, optional): Midpoint percentage for conversion; calculated if omitted.
*   **Response (201 Created):
    *   `id` (string): The UUID of the newly created profile.
    *   `range_ids` (array of strings): UUIDs of the created ranges.

### 2. List Grading Scale Profiles

*   **URL:** `/api/v1/grading/profiles`
*   **Method:** `GET`
*   **Description:** Lists all grading scale profiles for the current school.
*   **Authentication:** Required
*   **Query Parameters:**
    *   `active_only` (boolean, optional): If `true`, only returns active profiles. Defaults to `false`.
*   **Response (ListScaleProfilesResponse):
    *   `items` (array of objects):
        *   `id` (string)
        *   `name` (string)
        *   `is_active` (boolean)
        *   `created_at` (string, ISO 8601)
        *   `updated_at` (string, ISO 8601)
    *   `total` (integer)
    *   `page` (integer)
    *   `limit` (integer)

### 3. Get Grading Scale Profile by ID

*   **URL:** `/api/v1/grading/profiles/:id`
*   **Method:** `GET`
*   **Description:** Retrieves a single grading scale profile by its ID, including its nested percentage-to-level ranges.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the profile.
*   **Response (Profile):** A single `Profile` object with an embedded `ranges` array.

### 4. Get Grading Scale Ranges

*   **URL:** `/api/v1/grading/profiles/:id/ranges`
*   **Method:** `GET`
*   **Description:** Returns all percentage-to-level ranges for a specific grading scale profile.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the profile.
*   **Response:**
    *   `items` (array of objects):
        *   `id` (string)
        *   `profile_id` (string)
        *   `performance_level` (string)
        *   `min_percentage` (integer)
        *   `max_percentage` (integer)
        *   `default_percentage_mapping` (integer, optional)

### 5. Replace Grading Scale Ranges

*   **URL:** `/api/v1/grading/profiles/:id/ranges`
*   **Method:** `PUT`
*   **Description:** Replaces all existing ranges for a grading scale profile with a new set. This is an atomic operation. Requires at least EE, ME, and AE ranges.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the profile.
*   **Request Body:**
    *   `ranges` (array of ScaleRangePayload): Same structure as `CreateScaleProfilePayload.ranges`.
*   **Response:**
    *   `ids` (array of strings): UUIDs of the newly created ranges.

### 6. Toggle Grading Scale Profile Active Status

*   **URL:** `/api/v1/grading/profiles/:id/toggle`
*   **Method:** `PUT`
*   **Description:** Toggles the `is_active` flag for a grading scale profile. Used to deprecate profiles while retaining historical data.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the profile.
*   **Query Parameters:**
    *   `is_active` (boolean, optional): Set to `false` to deactivate. Defaults to `true`.
*   **Response:**
    *   `message` (string): "profile updated"

### 7. Delete Grading Scale Profile

*   **URL:** `/api/v1/grading/profiles`
*   **Method:** `DELETE`
*   **Description:** Permanently deletes a grading scale profile and its ranges. This operation is blocked if any assessment session references the profile.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Request Body:**
    *   `id` (string): The ID of the profile to delete.
*   **Response:** `204 No Content`

## Assessment Sessions

### 8. Create Assessment Session

*   **URL:** `/api/v1/assessments/sessions`
*   **Method:** `POST`
*   **Description:** Creates a new assessment session in `DRAFT` status. Supports `QUANTITATIVE` (score-based) or `RUBRIC` (indicator-based) evaluation methods.
*   **Authentication:** Required (Teacher role implicitly)
*   **Request Body (CreateSessionPayload):
    *   `class_id` (string, required)
    *   `learning_area_id` (string, required)
    *   `academic_term_id` (string, required)
    *   `academic_year_id` (string, required)
    *   `name` (string, required)
    *   `evaluation_method` (string, required): "QUANTITATIVE" or "RUBRIC".
    *   `max_points` (integer, optional): Required for `QUANTITATIVE`, omit for `RUBRIC`.
    *   `grading_scale_profile_id` (string, optional): Required for `QUANTITATIVE`, omit for `RUBRIC`.
    *   `scheduled_date` (string, YYYY-MM-DD, optional)
*   **Response (201 Created):
    *   `id` (string): The UUID of the new session.

### 9. Get Grading Data for Session

*   **URL:** `/api/v1/assessments/sessions/:id/grading-data`
*   **Method:** `GET`
*   **Description:** Returns session details, class roster, and existing scores/grades merged into a single response.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session.
*   **Response (GradingDataResponse):
    *   `session` (object): Detailed session information.
    *   `students` (array of objects): Each student includes `student_id`, `student_name`, `admission_number`, `gender`, `enrollment_status`.
        *   For `QUANTITATIVE` sessions, `score` (object: `raw_score`, `calculated_percentage`, `final_performance_level`).
        *   For `RUBRIC` sessions, `grades` (array of objects: `performance_indicator_id`, `awarded_level`).

### 10. Get Assessment Session by ID

*   **URL:** `/api/v1/assessments/sessions/:id`
*   **Method:** `GET`
*   **Description:** Retrieves a single assessment session by ID, including its status, evaluation method, learning area, and audit trail.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session.
*   **Response (Session):** A detailed `Session` object.

### 11. List Assessment Sessions

*   **URL:** `/api/v1/assessments/sessions`
*   **Method:** `GET`
*   **Description:** Provides a paginated list of assessment sessions with various filtering options.
*   **Authentication:** Required
*   **Query Parameters (SessionFilters):
    *   `class_id` (string, optional)
    *   `learning_area_id` (string, optional)
    *   `academic_term_id` (string, optional)
    *   `status` (string, optional): "DRAFT", "PENDING_APPROVAL", "PUBLISHED".
    *   `evaluation_method` (string, optional): "QUANTITATIVE", "RUBRIC".
    *   `search` (string, optional): Fuzzy match on session name.
    *   `page` (integer, optional): Default 1.
    *   `limit` (integer, optional): Default 50, max 100.
*   **Response (ListSessionsResponse):
    *   `items` (array of Session): List of `Session` objects.
    *   `total` (integer)
    *   `page` (integer)
    *   `limit` (integer)

### 12. Delete Assessment Session

*   **URL:** `/api/v1/assessments/sessions`
*   **Method:** `DELETE`
*   **Description:** Permanently deletes a `DRAFT` assessment session and its associated scores/grades. Cannot delete sessions in `PENDING_APPROVAL` or `PUBLISHED` status.
*   **Authentication:** Required
*   **Request Body:**
    *   `id` (string): The ID of the session to delete.
*   **Response:** `204 No Content`

### 13. Submit Assessment Session

*   **URL:** `/api/v1/assessments/sessions/:id/submit`
*   **Method:** `POST`
*   **Description:** Transitions a session from `DRAFT` to `PENDING_APPROVAL`, locking it from further teacher edits. Clears any existing rejection comments.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session to submit.
*   **Response:**
    *   `message` (string): "session submitted for approval"

### 14. Approve Assessment Session

*   **URL:** `/api/v1/assessments/sessions/:id/approve`
*   **Method:** `POST`
*   **Description:** Transitions a session from `PENDING_APPROVAL` to `PUBLISHED`. For `QUANTITATIVE` sessions, it computes and snapshots CBC levels (EE/ME/AE/BE) from percentages.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session to approve.
*   **Response:**
    *   `message` (string): "session approved and published"

### 15. Reject Assessment Session

*   **URL:** `/api/v1/assessments/sessions/:id/reject`
*   **Method:** `POST`
*   **Description:** Transitions a session from `PENDING_APPROVAL` back to `DRAFT`. Requires a `rejection_comment`. The session becomes editable again.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session to reject.
*   **Request Body (RejectSessionPayload):
    *   `rejection_comment` (string, required): Explanation for the rejection.
*   **Response:**
    *   `message` (string): "session rejected and returned to draft"

## Student Scores (Quantitative)

### 16. Bulk Upsert Student Scores

*   **URL:** `/api/v1/assessments/sessions/:id/scores`
*   **Method:** `POST`
*   **Description:** Bulk-upserts quantitative raw scores for students in a `QUANTITATIVE` session. Scores can only be modified in `DRAFT` status. Students marked `ABSENT` or `EXEMPT` cannot receive scores.
*   **Authentication:** Required (Teacher role implicitly)
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session.
*   **Request Body (BulkUpsertScoresPayload):
    *   `scores` (array of objects):
        *   `student_id` (string, required)
        *   `raw_score` (integer, optional): The student's raw score. Omit or set to `null` if not graded.
*   **Response:**
    *   `message` (string): "scores saved"

### 17. Get Student Scores

*   **URL:** `/api/v1/assessments/sessions/:id/scores`
*   **Method:** `GET`
*   **Description:** Returns all quantitative scores for a session, including calculated percentage and final performance level (if published).
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session.
*   **Response (StudentScoresResponse):
    *   `items` (array of objects):
        *   `id` (string)
        *   `session_id` (string)
        *   `student_id` (string)
        *   `raw_score` (integer)
        *   `calculated_percentage` (float)
        *   `final_performance_level` (string, null before approval)
        *   `enrollment_status` (string)

## Student Outcome Grades (Rubric)

### 18. Bulk Upsert Outcome Grades

*   **URL:** `/api/v1/assessments/sessions/:id/grades`
*   **Method:** `POST`
*   **Description:** Bulk-upserts rubric outcome grades for a `RUBRIC` session. Teachers assign CBC levels directly per performance indicator. Only modifiable in `DRAFT` status.
*   **Authentication:** Required (Teacher role implicitly)
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session.
*   **Request Body (BulkUpsertOutcomeGradesPayload):
    *   `grades` (array of objects):
        *   `student_id` (string, required)
        *   `performance_indicator_id` (string, required)
        *   `awarded_level` (string, required): "EE", "ME", "AE", "BE".
*   **Response:**
    *   `message` (string): "grades saved"

### 19. Get Outcome Grades

*   **URL:** `/api/v1/assessments/sessions/:id/grades`
*   **Method:** `GET`
*   **Description:** Returns all rubric outcome grades for a `RUBRIC` session, showing each student's awarded level per performance indicator.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session.
*   **Response (OutcomeGradesResponse):
    *   `items` (array of objects):
        *   `id` (string)
        *   `session_id` (string)
        *   `student_id` (string)
        *   `performance_indicator_id` (string)
        *   `awarded_level` (string)

## Parent View & Report Cards

### 20. Get Parent Assessments

*   **URL:** `/api/v1/parent/students/:studentId/assessments`
*   **Method:** `GET`
*   **Description:** Returns all `PUBLISHED` assessment results for a specific student within a given academic term. This is the real-time view for parents.
*   **Authentication:** Required (Parent role implicitly)
*   **Path Parameters:**
    *   `studentId` (string)
*   **Query Parameters:**
    *   `academic_term_id` (string, required)
*   **Response (ParentTermAssessmentsResponse):
    *   `items` (array of objects):
        *   `session_id` (string)
        *   `session_name` (string)
        *   `evaluation_method` (string)
        *   `scheduled_date` (string, YYYY-MM-DD)
        *   For `QUANTITATIVE`: `raw_score`, `max_points`, `performance_level`.
        *   For `RUBRIC`: `outcome_grades` (array of objects: `performance_indicator_id`, `awarded_level`).

### 21. Get Student Term Grades (Report Card)

*   **URL:** `/api/v1/parent/students/:studentId/report-card`
*   **Method:** `GET`
*   **Description:** Compiles the end-of-term report card for a student using the "Last One" Chronological Mode aggregator to determine final grades per learning area.
*   **Authentication:** Required (Parent role implicitly)
*   **Path Parameters:**
    *   `studentId` (string)
*   **Query Parameters:**
    *   `academic_term_id` (string, required)
*   **Response (StudentTermGradesResponse):
    *   `items` (array of objects):
        *   `learning_area_id` (string)
        *   `learning_area_name` (string)
        *   `learning_area_code` (string)
        *   `final_level` (string): The aggregated CBC level (EE/ME/AE/BE).
        *   `assessment_count` (integer): Number of assessments contributing to this grade.

## Student Term Subject Summaries

### 22. Get Student Term Subject Summaries

*   **URL:** `/api/v1/parent/students/:studentId/term-subject-summaries`
*   **Method:** `GET`
*   **Description:** Returns blended assessment summaries for a student across all learning areas in a given term. Includes average percentage, mapped performance level, and source counts.
*   **Authentication:** Required (Parent role implicitly)
*   **Path Parameters:**
    *   `studentId` (string)
*   **Query Parameters:**
    *   `academic_term_id` (string, required)
*   **Response (StudentTermSubjectSummariesResponse):
    *   `items` (array of objects):
        *   `student_id` (string)
        *   `academic_term_id` (string)
        *   `learning_area_id` (string)
        *   `average_percentage` (float)
        *   `mapped_performance_level` (string)
        *   `quantitative_assessment_count` (integer)
        *   `rubric_assessment_count` (integer)
        *   `indicators_assessed_count` (integer)
        *   `has_quantitative_data` (boolean)
        *   `has_rubric_data` (boolean)
        *   `teacher_remark` (string, optional)
        *   `last_refreshed_at` (string, ISO 8601)

### 23. Get Learning Area Summaries (Teacher Dashboard)

*   **URL:** `/api/v1/assessments/sessions/learning-area/:learningAreaId/term-subject-summaries`
*   **Method:** `GET`
*   **Description:** Returns summaries for all students in a specific learning area for a given term, useful for teacher dashboards.
*   **Authentication:** Required (Teacher role implicitly)
*   **Path Parameters:**
    *   `learningAreaId` (string)
*   **Query Parameters:**
    *   `academic_term_id` (string, required)
*   **Response (StudentTermSubjectSummariesResponse):** Same structure as "Get Student Term Subject Summaries", but for multiple students.

### 24. Refresh Term Subject Summary

*   **URL:** `/api/v1/assessments/term-subject-summaries/refresh`
*   **Method:** `POST`
*   **Description:** Manually triggers a refresh of student term subject summaries for a given session.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Request Body:
    *   `session_id` (string, required)
*   **Response:**
    *   `message` (string): "summary refreshed"

### 25. Set Teacher Remark on Summary

*   **URL:** `/api/v1/assessments/term-subject-summaries/:id/remark`
*   **Method:** `PUT`
*   **Description:** Sets or clears a teacher's remark on a specific student term subject summary row.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the summary row.
*   **Request Body (SetTeacherRemarkPayload):
    *   `remark` (string, optional): The remark text. Pass `null` to clear.
*   **Response:**
    *   `message` (string): "remark updated"

## Assessment Weight Configs

### 26. Create Assessment Weight Config

*   **URL:** `/api/v1/assessments/weight-configs`
*   **Method:** `POST`
*   **Description:** Creates a new KNEC weight configuration entry, defining how assessment types contribute to a target exam placement score. These are system-level configs.
*   **Authentication:** Required (SYSTEM_ADMIN role)
*   **Request Body (CreateWeightConfigPayload):
    *   `grade_level` (string, required): E.g., "GRADE_4".
    *   `assessment_type_code` (string, required): E.g., "KNEC_SBA_Project".
    *   `target_exam` (string, required): E.g., "KPSEA".
    *   `weight_percent` (float, required): The percentage weight.
    *   `effective_from` (integer, required): The academic year from which this config is effective.
    *   `notes` (string, optional): Additional notes.
*   **Response (201 Created):
    *   `id` (string): UUID of the new weight config.

### 27. List Assessment Weight Configs

*   **URL:** `/api/v1/assessments/weight-configs`
*   **Method:** `GET`
*   **Description:** Lists all assessment weight configurations, with optional filters.
*   **Authentication:** Required
*   **Query Parameters (AssessmentWeightConfigFilter):
    *   `grade_level` (string, optional)
    *   `target_exam` (string, optional)
    *   `effective_from` (integer, optional)
*   **Response (List of WeightConfig):** An array of `WeightConfig` objects.

### 28. Get Assessment Weight Config by ID

*   **URL:** `/api/v1/assessments/weight-configs/:id`
*   **Method:** `GET`
*   **Description:** Retrieves a single assessment weight configuration by its ID.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the weight config.
*   **Response (WeightConfig):** A single `WeightConfig` object.

### 29. Delete Assessment Weight Config

*   **URL:** `/api/v1/assessments/weight-configs`
*   **Method:** `DELETE`
*   **Description:** Permanently deletes an assessment weight configuration.
*   **Authentication:** Required (SYSTEM_ADMIN role)
*   **Request Body:
    *   `id` (string): The ID of the weight config to delete.
*   **Response:** `204 No Content`

## Student Term Overall Summaries

### 30. Refresh Term Overall Summaries (All Students)

*   **URL:** `/api/v1/assessments/term-overall-summaries/refresh`
*   **Method:** `POST`
*   **Description:** Triggers the computation of overall summaries for all students in a given term.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Request Body:
    *   `term_id` (string, required)
*   **Response:**
    *   `message` (string): "overall summaries refreshed"

### 31. Refresh Single Student Overall Summary

*   **URL:** `/api/v1/assessments/term-overall-summaries/refresh-student`
*   **Method:** `POST`
*   **Description:** Triggers the computation of the overall summary for a single student-term pair. Useful after subject summary updates.
*   **Authentication:** Required
*   **Request Body:
    *   `student_id` (string, required)
    *   `term_id` (string, required)
*   **Response:**
    *   `message` (string): "student overall summary refreshed"

### 32. Get Student Term Overall Summary

*   **URL:** `/api/v1/assessments/term-overall-summaries/:studentId/:termId`
*   **Method:** `GET`
*   **Description:** Returns the overall summary for a single student-term pair.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `studentId` (string)
    *   `termId` (string)
*   **Response (StudentTermOverallSummaryResponse):
    *   `data` (object): A `StudentTermOverallSummary` object. (Includes aggregated performance data, final grades, etc.)

### 33. List Student Term Overall Summaries

*   **URL:** `/api/v1/assessments/term-overall-summaries`
*   **Method:** `GET`
*   **Description:** Returns overall summaries for all students in the given term.
*   **Authentication:** Required
*   **Query Parameters:**
    *   `term_id` (string, required)
*   **Response (StudentTermOverallSummariesListResponse):
    *   `items` (array of StudentTermOverallSummary): List of summary objects.

### 34. Set Headteacher Remark on Overall Summary

*   **URL:** `/api/v1/assessments/term-overall-summaries/:id/headteacher-remark`
*   **Method:** `PUT`
*   **Description:** Sets or clears the headteacher's remark on an overall summary row.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the overall summary row.
*   **Request Body (SetHeadteacherRemarkPayload):
    *   `remark` (string, optional): The remark text. Pass `null` to clear.
*   **Response:**
    *   `message` (string): "headteacher remark updated"

## Student Subject Strand Summaries

### 35. Refresh Subject Strand Summaries

*   **URL:** `/api/v1/assessments/subject-strand-summaries/refresh`
*   **Method:** `POST`
*   **Description:** Triggers a refresh of sub-strand summaries for a given session.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Request Body:
    *   `session_id` (string, required)
*   **Response:**
    *   `message` (string): "sub-strand summaries refresh initiated"

### 36. Get Student Subject Strand Summaries

*   **URL:** `/api/v1/assessments/subject-strand-summaries/:studentId/:termId`
*   **Method:** `GET`
*   **Description:** Returns sub-strand summaries for a specific student and term. These are rubric-only, sub-strand level summaries.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `studentId` (string)
    *   `termId` (string)
*   **Response (StudentSubjectStrandSummariesResponse):
    *   `items` (array of objects): List of `StudentSubjectStrandSummary` objects.
    *   `total` (integer)

### 37. List Subject Strand Summaries By Term

*   **URL:** `/api/v1/assessments/subject-strand-summaries`
*   **Method:** `GET`
*   **Description:** Returns all sub-strand summaries for a given term.
*   **Authentication:** Required
*   **Query Parameters:**
    *   `term_id` (string, required)
*   **Response (StudentSubjectStrandSummariesResponse):
    *   `items` (array of objects): List of `StudentSubjectStrandSummary` objects.
    *   `total` (integer)

## Student Performance Projections

### 38. Refresh Performance Projections

*   **URL:** `/api/v1/assessments/projections/refresh`
*   **Method:** `POST`
*   **Description:** Triggers a batch refresh of performance projections for a given term.
*   **Authentication:** Required (SCHOOL_ADMIN role)
*   **Request Body (RefreshProjectionsRequest):
    *   `academic_term_id` (string, required)
*   **Response (RefreshProjectionsResponse):
    *   `message` (string): "performance projections refresh initiated"
    *   `term_id` (string)

### 39. Get Student Performance Projection

*   **URL:** `/api/v1/assessments/projections/:studentId/:termId`
*   **Method:** `GET`
*   **Description:** Returns the performance projection for a specific student-term pair, optionally filtered by learning area.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `studentId` (string)
    *   `termId` (string)
*   **Query Parameters:**
    *   `learning_area_id` (string, optional)
*   **Response (PerformanceProjectionResponse):
    *   `data` (object): A `PerformanceProjection` object.

### 40. List Student Performance Projections

*   **URL:** `/api/v1/assessments/projections`
*   **Method:** `GET`
*   **Description:** Returns all performance projections for a given term.
*   **Authentication:** Required
*   **Query Parameters:**
    *   `term_id` (string, required)
*   **Response (PerformanceProjectionListResponse):
    *   `items` (array of PerformanceProjection): List of projection objects.
    *   `total` (integer)
