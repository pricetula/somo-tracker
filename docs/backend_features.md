# Backend Features Documentation

This document outlines the completed features identified within the `backend/internal` directory of the Somotracker project. The presence of handler, service, and repository files within each module is used as an indicator of feature implementation.

## Implemented Features

### Academic Years
**Description:** Manages the definition and lifecycle of academic years within the system.
**Path:** `backend/internal/academicyears/`

### Assessments
**Description:** Handles the creation, management, and tracking of student assessments.
**Path:** `backend/internal/assessments/`

### Attendance
**Description:** Manages student and teacher attendance records.
**Path:** `backend/internal/attendance/`

### Authentication (Auth)
**Description:** Provides user authentication and authorization functionalities, likely integrating with Stytch based on observed files.
**Path:** `backend/internal/auth/`

### Behavior
**Description:** Manages student behavior records and related functionalities.
**Path:** `backend/internal/behavior/`

### Billing
**Description:** Handles billing-related operations, such as invoicing and payment tracking.
**Path:** `backend/internal/billing/`

### CBC Classes
**Description:** Manages classes related to the Competency-Based Curriculum (CBC) framework.
**Path:** `backend/internal/cbcclasses/`

### CBC Schools
**Description:** Manages school-specific data within the CBC framework.
**Path:** `backend/internal/cbcschools/`

### CBC Streams
**Description:** Manages academic streams or tracks within the CBC framework.
**Path:** `backend/internal/cbcstreams/`

### CBC Timetable Slots
**Description:** Manages timetable slots specific to the CBC curriculum.
**Path:** `backend/internal/cbctimetableslots/`

### Class Teachers
**Description:** Manages the assignment and data of teachers to specific classes.
**Path:** `backend/internal/classteachers/`

### Cohort Positions
**Description:** Manages positions or roles within student cohorts.
**Path:** `backend/internal/cohortpositions/`

### Configuration
**Description:** Handles application configuration settings.
**Path:** `backend/internal/config/`

### Curriculum
**Description:** Manages curriculum-related data and structures.
**Path:** `backend/internal/curriculum/`

### Database
**Description:** Contains database migration, connection, and utility functions.
**Path:** `backend/internal/database/`

### Health Check
**Description:** Provides endpoints for application health monitoring.
**Path:** `backend/internal/health/`

### Imports
**Description:** Manages data import processes into the system.
**Path:** `backend/internal/imports/`

### Invitations
**Description:** Handles user invitation functionalities.
**Path:** `backend/internal/invitations/`

### Members
**Description:** Manages general member (user) data and roles.
**Path:** `backend/internal/members/`

### Middleware
**Description:** Contains HTTP middleware for request processing (e.g., error handling, authentication checks).
**Path:** `backend/internal/middleware/`

### Parents
**Description:** Manages parent accounts and their associated student data.
**Path:** `backend/internal/parents/`

### Reports
**Description:** Generates and manages various reports within the system.
**Path:** `backend/internal/reports/`

### Resources
**Description:** Manages static or dynamic resources used by the application.
**Path:** `backend/internal/resources/`

### Slug Generation
**Description:** Provides functionality for generating URL-friendly slugs.
**Path:** `backend/internal/slug/`

### Students
**Description:** Manages student profiles, enrollment, and related data.
**Path:** `backend/internal/students/`

### Teacher Delivery Summaries
**Description:** Manages summaries of teacher-delivered content or lessons.
**Path:** `backend/internal/teacherdeliverysummaries/`

### Teacher Performance
**Description:** Tracks and manages teacher performance data.
**Path:** `backend/internal/teacherperformance/`

### Teachers
**Description:** Manages teacher profiles, assignments, and related data.
**Path:** `backend/internal/teachers/`

### Teacher Workload Summaries
**Description:** Manages summaries of teacher workload.
**Path:** `backend/internal/teacherworkloadsummaries/`

### Timetable Structure
**Description:** Defines and manages the overall structure of timetables.
**Path:** `backend/internal/timetablestructure/`

### Utilities
**Description:** Contains general utility functions used across the backend.
**Path:** `backend/internal/utils/`
