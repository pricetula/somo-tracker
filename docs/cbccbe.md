# SomoTracker Academic Tracking Layer: CBC/CBE Assessment & Reporting Framework

## Educator & School Administrator Reference Manual

This guide provides a comprehensive, non-technical overview of how **SomoTracker** manages assessments under the Kenyan Competency-Based Curriculum (CBC) / Competency-Based Education (CBE) framework for the 2-6-3-3 system.

It maps our behind-the-scenes application data tables directly to daily classroom tasks, showing teachers and school administrators exactly how data flows from a lesson plan all the way to end-of-term reporting and the manual KNEC portal updates.

## 1\. The Core Infrastructure: Our System Tables

To keep SomoTracker lightweight, secure, and compliant with the Ministry of Education and KICD guidelines, your school's data is organized into a structural hierarchy. Here are the core building blocks:

### 1.1 The Curriculum Blueprint Tables

These tables are pre-seeded and managed by the system based on official KICD design sheets. Teachers do not need to build these; they simply select from them.

- **cbc_learning_areas**: Stores the official subjects taught per grade level (e.g., _Mathematical Activities_, _Language Activities_, or _Agriculture & Nutrition_).
- **cbc_strands**: The main thematic units within a subject (e.g., _Numbers_, _Geometry_, or _Crop Production_).
- **cbc_sub_strands**: The sub-topics or specific sub-units being taught (e.g., _Fractions_, or _Soil Preparation_).
- **performance_indicators**: The atomic, leaf-node measurable skills defined by KICD (e.g., _"Can identify safety wear during soil preparation"_). This is the exact skill level at which students are evaluated.

### 1.2 The School & Learner Operational Tables

These tables maintain your school’s unique structure and secure government records.

- **cbc_schools**: Contains your school’s core metadata, strictly requiring key national identification codes like the **knec_school_code** (your official examination center code used as a username) and the **nemis_institution_code**.
- **cbc_students**: Holds individual student profiles. For government regulatory compliance, each student record includes their official **upi_number** (Unique Personal Identifier), **knec_assessment_number**, and assigned **learning_pathway**.

### 1.3 The Active Assessment & Result Tables

These tables capture daily academic activity and compile performance records.

- **assessment_blueprints**: The master template created by a teacher for a quiz, project, or exam. It outlines the topic, term, year, and assessment instrument type.
- **assessment_blueprint_indicators**: A hidden connector table that maps a single assessment blueprint to one or more precise KICD performance_indicators.
- **assessment_sessions**: Represents the physical execution of an assessment template for a specific stream on a given date.
- **learner_rubric_results**: The atomic record where an individual student’s outcome is stored. It holds raw tracking values (raw_score) or direct rubric evaluations (rubric_level).
- **learner_portfolios**: Stores digital evidence references (like photos or media URLs) linked to specific rubric entries.
- **cbc_term_competency_summaries**: The final repository table where individual outcomes are weighted and consolidated into a definitive term-end evaluation used for report cards and portal transcription.

## 2\. Strict Core Rules for Assessment Data

To maintain absolute CBC compliance, the system operates under rigid data boundaries:

- **No Class Ranks or Positions:** The app completely rejects overall averages, positional metrics, or rank indexes.
- **No Percentage Grades:** Students are never assigned percentages (e.g., 72%).
- **Four-Value Rubric Matrix:** The primary scale of academic achievement is strictly constrained to the official KICD descriptive metrics:
  - **EE**: Exceeds Expectations
  - **ME**: Meets Expectations
  - **AE**: Approaching Expectations
  - **BE**: Below Expectations

- **Non-Cumulative Numbers:** The system allows a numeric raw_score on individual tasks (e.g., scoring a classroom quiz out of 10 points), but these points are solely used to determine a rubric level. They are **never** averaged together across different tests or sub-strands.

## 3\. End-to-End Workflows

### Section A: Classroom Formative Assessments

Formative assessments are daily ongoing diagnostics used purely for internal growth tracking. They do not get uploaded to the government.

#### Workflow 1: Designing a Formative Assessment

1.  A teacher navigates to the assessment setup panel to build a classroom task.
2.  The teacher provides descriptive metadata, creating a record in **assessment_blueprints**:
    - **Title**: Week 3 Addition & Subtraction Check
    - **Type**: Formative_Classroom
    - **Grade Level**: G3
    - **Term**: 1 | **Academic Year**: 2026

3.  The system presents the teacher with a checkbox tree of KICD milestones derived from **cbc_sub_strands**. The teacher selects the target outcomes, saving matching entries to **assessment_blueprint_indicators** linked to the target **performance_indicators**.

#### Workflow 2: Executing the Assessment and Data Entry

1.  On the day of the activity, the teacher opens their stream roster, creating an active row in **assessment_sessions**. The date_administered field is locked explicitly to that day's date, and the record references the teacher’s profile.
2.  As the class interacts with the activity, the teacher enters evaluations for each student, writing directly to **learner_rubric_results**:
    - **Direct Rubric Entry**: For oral tasks, the teacher taps a button to assign a direct level (e.g., ME or AE).
    - **Raw Score Entry**: For short written exercises marked out of a small score, the teacher inputs a numeric mark (e.g., 7.00). The underlying application logic looks up the scaling parameters to flag the corresponding level, but does not alter other academic fields.

3.  If a student demonstrates an impressive physical skill (e.g., physical education or music performance), the teacher can capture a quick digital snapshot, creating an entry in **learner_portfolios** pinned to that specific row in learner_rubric_results.

### Section B: Official KNEC Summative Assessments

Summative assessments are standardized assessments mandated by the national government at specific structural milestones.

#### Workflow 1: Initializing a KNEC Assessment Template

1.  During Term 3 for upper grades, or when designated milestones occur, official templates are prepared. The school or system initializes a template row in **assessment_blueprints**.
2.  The type field is strictly defined based on the official instrument chosen:
    - KNEC_Written_Assessment (For standardized written papers)
    - KNEC_SBA_Project (For national school-based project tasks)
    - National_KPSEA / National_KJSEA / National_KSSEA (For the official curriculum transition exams)

3.  The underlying **assessment_blueprint_indicators** are matched precisely to the official standardized rubrics provided by KNEC.

#### Workflow 2: Administering and Recording Summative Outcomes

1.  The teacher administers the standardized paper or guides the students through the physical assessment project.
2.  A row is logged in **assessment_sessions**. Crucially, the execution date (date_administered) contains **no default value** and must be populated manually by the educator, allowing proper logging for multi-day national projects.
3.  The teacher marks the student submissions. The individual project parameters are saved inside **learner_rubric_results**.
4.  At the end of the semester, the system executes an evaluation pass. It pulls data from all relevant sessions and applies the official weightings defined in the system's core parameters. This populates a clean, finalized row in **cbc_term_competency_summaries** containing the student's **calculated_level** and **final_level**.

## 4\. Operational Data Previews

These simplified data views illustrate exactly how information flows through the system tables during a term.

### 4.1 Sample: Assessment Setup Data (assessment_blueprints)

This shows how different assessment instruments are organized inside the app before execution.

**Blueprint IDAcademic TitleInstrument Type (type)GradeTermAcademic YearTarget Subject (learning_area_id)**b-9901Grade 4 Agriculture ProjectKNEC_SBA_ProjectG422026Agriculture & Nutritionb-9902Week 3 Reading CheckFormative_ClassroomG112026Language Activities

### 4.2 Sample: Active Evaluation Records (learner_rubric_results)

This shows individual student scores captured against specific KICD performance milestones during an assessment session.

**Session ReferenceStudent NameNational ID Lookup (knec_assessment_number)Specific Milestone Tested (indicator_id)Score SetupRaw MarkEvaluated Level (rubric_level)**sess-9901 (Agri Project)John MwangiA-1029384\*Safe tool handling_Numeric Raw18.00 / 20**ME** (Meets)sess-9901 (Agri Project)Amina OchiengA-5839201_Safe tool handling_Direct Rubric_None**\*EE** (Exceeds)sess-9902 (Reading Check)David Kiprop\*None (Lower Primary)Three-letter word fluidity_Direct Rubric_None**\*AE** (Approaching)

### 4.3 Sample: Finalized Term Compilation (cbc_term_competency_summaries)

This data matrix serves as the direct reference source used to fill out student report cards and guide manual data transcription into the government web systems.

**Student NameSubjectYear / TermCalculated Score TierAuthorized OverrideDefinitive Grade (final_level)Portal Upload Flag (knec_sync_status)John Mwangi**Agriculture2026, Term 2ME\*None**\*ME**⭕ Pending Manual Entry**Amina Ochieng**Agriculture2026, Term 2MEEE _(Teacher Adjusted)_**EE**⭕ Pending Manual Entry**David Kiprop**Agriculture2026, Term 2AE\*None**\*AE**Green Checkmark Copy Done

## 5\. Interaction Flow: SomoTracker ◄► KNEC CBA Portal

Because the official KNEC CBA Portal (cba.knec.ac.ke) is an isolated government system without public integrations or API channels, automated background data syncing is impossible.

SomoTracker bridges this gap by acting as a "Preparation Sandbox." The app cleans, weights, and organizes all internal evaluation rows so they line up perfectly with the manual data entry layouts on the government portal.

```
+------------------------------------------------------------+
| STEP 1: SOMOTRACKER EXPORT |
| Teacher opens the "KNEC Portal Sync Dashboard" in the app, |
| filtering by Term, Grade, and Learning Area. |
+------------------------------------------------------------+
│
▼
+------------------------------------------------------------+
| STEP 2: RECORD PREPARATION |
| The screen displays each student's official Name, their |
| KNEC Assessment Number, and their verified Final Level. |
+------------------------------------------------------------+
│
▼
+------------------------------------------------------------+
| STEP 3: SECURE GOVERNMENT PORTAL LOG IN |
| The teacher opens http://cba.knec.ac.ke in a browser split- |
| screen and logs in using the school's unique Center Code. |
+------------------------------------------------------------+
│
▼
+------------------------------------------------------------+
| STEP 4: TARGETED DATA ENTRY |
| The teacher locates the student on the KNEC site using the |
| Assessment Number and copies over the Final Level (EE/ME/etc).|
+------------------------------------------------------------+
│
▼
+------------------------------------------------------------+
| STEP 5: CLOSING THE AUDIT TRAIL |
| After saving on the KNEC site, the teacher enters the web |
| confirmation receipt into SomoTracker to change the status |
| from "Pending Manual Entry" to "Synced". |
+------------------------------------------------------------+
```

### Why This Workflow Protects Your School

By organizing and locking down final term calculations inside **cbc_term_competency_summaries** before data entry begins, SomoTracker ensures that the school has a bulletproof, unalterable audit trail. It eliminates transcription mistakes, removes the need for temporary paper score sheets, and speeds up manual data submission during peak end-of-term periods.
