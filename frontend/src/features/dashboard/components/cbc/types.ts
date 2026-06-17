// ── Section 1A: Term Banner ──
export type TermBannerData =
    | { state: "active"; termName: string; yearName: string; startDate: string; endDate: string }
    | { state: "no-term"; yearName: string }
    | { state: "no-config" }
    | { state: "out-of-range" };

// ── Section 1B: Enrollment ──
export type EnrollmentData = { active: number; suspended: number; transferred: number };

// ── Section 1C: Daily Attendance ──
export type AttendanceData = { present: number; late: number; absent: number; total: number };

// ── Section 1D: Health Flags ──
export type HealthFlagStudent = {
    name: string;
    class: string;
    conditions: string[];
    emergency: string;
};
export type HealthFlagsData = { count: number; students: HealthFlagStudent[] };

// ── Section 2A: Performance Matrix ──
export type ScoreLevel = "EE" | "ME" | "AE" | "BE";
export type StudentRow = {
    rank: number;
    name: string;
    class: string;
    avgScore: number;
    scoreLevel: ScoreLevel;
    delta: number;
};
export type InterventionRow = {
    name: string;
    class: string;
    beAreaCount: number;
    areas: string[];
};

// ── Section 2B-1: Mastery ──
export type MasteryRow = { area: string; EE: number; ME: number; AE: number; BE: number };

// ── Section 2B-2: Stream Performance ──
export type StreamRow = { level: ScoreLevel; east: number; west: number };

// ── Section 2B-3: YoY ──
export type YoYRow = { week: string; thisYear: number; lastYear: number };

// ── Section 2B-4: Scatter ──
export type ScatterPoint = { attendance: number; score: number };

// ── Section 2C: Compliance ──
export type ComplianceItem = {
    id: string;
    issue: string;
    count: number;
    severity: "error" | "warning";
    resolveLabel: string;
};

// ── Section 2D-1: Eval Compliance ──
export type EvalCompliance = {
    teacher: string;
    class: string;
    graded: number;
    total: number;
};

// ── Section 2D-2: Weighting ──
export type WeightingIssue = {
    class: string;
    issue: string;
    severity: "error" | "warning";
};

// ── Section 2D-3: Absentee ──
export type AbsenteeRow = {
    name: string;
    class: string;
    absentDays: number;
    threshold: number;
};

// ── Section 2E: Finance ──
export type FeeCategory = { name: string; configured: boolean };
export type FinanceData = {
    outstandingTotal: number;
    invoiceCount: number;
    totalInvoiced: number;
    totalCollected: number;
    currency: string;
    feeCategories: FeeCategory[];
};

// ── Section 2F-1: Pending Users ──
export type PendingUser = {
    name: string;
    role: "TEACHER" | "SUPPORT_STAFF" | "SCHOOL_ADMIN";
    email: string;
    invitedAt: string;
};

// ── Section 2F-2: Medical Incidents ──
export type MedicalIncident = {
    student: string;
    class: string;
    time: string;
    symptoms: string;
    action: string;
};
