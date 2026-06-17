import type {
    TermBannerData,
    EnrollmentData,
    AttendanceData,
    HealthFlagsData,
    StudentRow,
    InterventionRow,
    MasteryRow,
    StreamRow,
    YoYRow,
    ScatterPoint,
    ComplianceItem,
    EvalCompliance,
    WeightingIssue,
    AbsenteeRow,
    FinanceData,
    PendingUser,
    MedicalIncident,
} from "./types";

// ── 1A: Term Banner ──
export const termBanner: TermBannerData = {
    state: "active",
    termName: "Term 2",
    yearName: "2026 Academic Year",
    startDate: "2026-05-05",
    endDate: "2026-08-15",
};

// ── 1B: Enrollment ──
export const enrollment: EnrollmentData = {
    active: 412,
    suspended: 8,
    transferred: 3,
};

// ── 1C: Daily Attendance ──
export const attendance: AttendanceData = {
    present: 341,
    late: 22,
    absent: 49,
    total: 412,
};

// ── 1D: Health Flags ──
export const healthFlags: HealthFlagsData = {
    count: 7,
    students: [
        {
            name: "Kevin Ochieng",
            class: "Grade 4 East",
            conditions: ["Asthma"],
            emergency: "Use blue inhaler. Call Dr. Otieno: 0722-xxx-xxx",
        },
        {
            name: "Mercy Wanjiku",
            class: "Grade 3 West",
            conditions: ["Epilepsy"],
            emergency: "Do not restrain. Time the seizure. Call parents immediately.",
        },
        {
            name: "James Kimani",
            class: "Grade 2 East",
            conditions: ["Severe nut allergy"],
            emergency: "EpiPen in staffroom fridge. Call 999.",
        },
    ],
};

// ── 2A: Performance Matrix ──
export const topPerformers: StudentRow[] = [
    {
        rank: 1,
        name: "Amina Wanjiru",
        class: "Grade 4 East",
        avgScore: 3.8,
        scoreLevel: "EE",
        delta: +0.4,
    },
    {
        rank: 2,
        name: "Brian Otieno",
        class: "Grade 3 West",
        avgScore: 3.6,
        scoreLevel: "EE",
        delta: +0.6,
    },
    {
        rank: 3,
        name: "Cynthia Achieng",
        class: "Grade 5 East",
        avgScore: 3.5,
        scoreLevel: "ME",
        delta: -0.2,
    },
    {
        rank: 4,
        name: "David Kamau",
        class: "Grade 4 West",
        avgScore: 3.4,
        scoreLevel: "ME",
        delta: +0.1,
    },
    {
        rank: 5,
        name: "Eunice Adhiambo",
        class: "Grade 2 East",
        avgScore: 3.3,
        scoreLevel: "ME",
        delta: +0.3,
    },
];

export const mostImproved: StudentRow[] = [
    {
        rank: 1,
        name: "Brian Otieno",
        class: "Grade 3 West",
        avgScore: 3.6,
        scoreLevel: "EE",
        delta: +0.6,
    },
    {
        rank: 2,
        name: "Amina Wanjiru",
        class: "Grade 4 East",
        avgScore: 3.8,
        scoreLevel: "EE",
        delta: +0.4,
    },
    {
        rank: 3,
        name: "Eunice Adhiambo",
        class: "Grade 2 East",
        avgScore: 3.3,
        scoreLevel: "ME",
        delta: +0.3,
    },
    {
        rank: 4,
        name: "Felix Muthoka",
        class: "Grade 5 West",
        avgScore: 2.9,
        scoreLevel: "AE",
        delta: +0.4,
    },
    {
        rank: 5,
        name: "David Kamau",
        class: "Grade 4 West",
        avgScore: 3.4,
        scoreLevel: "ME",
        delta: +0.1,
    },
];

export const regressed: StudentRow[] = [
    {
        rank: 1,
        name: "Grace Nyambura",
        class: "Grade 3 East",
        avgScore: 2.4,
        scoreLevel: "AE",
        delta: -0.8,
    },
    {
        rank: 2,
        name: "Henry Wekesa",
        class: "Grade 5 East",
        avgScore: 2.1,
        scoreLevel: "BE",
        delta: -0.6,
    },
    {
        rank: 3,
        name: "Isabel Mukami",
        class: "Grade 4 West",
        avgScore: 2.6,
        scoreLevel: "AE",
        delta: -0.4,
    },
    {
        rank: 4,
        name: "Cynthia Achieng",
        class: "Grade 5 East",
        avgScore: 3.5,
        scoreLevel: "ME",
        delta: -0.2,
    },
    {
        rank: 5,
        name: "Jack Onyango",
        class: "Grade 2 West",
        avgScore: 2.8,
        scoreLevel: "AE",
        delta: -0.1,
    },
];

export const interventionList: InterventionRow[] = [
    {
        name: "Faith Njeri",
        class: "Grade 3 East",
        beAreaCount: 5,
        areas: ["Literacy", "Numeracy", "CRE", "Science", "Social Studies"],
    },
    {
        name: "George Mwenda",
        class: "Grade 2 West",
        beAreaCount: 4,
        areas: ["Literacy", "Numeracy", "Art", "Music"],
    },
    {
        name: "Hannah Chebet",
        class: "Grade 4 East",
        beAreaCount: 3,
        areas: ["Numeracy", "Science", "Movement"],
    },
];

// ── 2B-1: Learning Area Mastery ──
export const masteryData: MasteryRow[] = [
    { area: "Mathematical", EE: 61, ME: 24, AE: 11, BE: 4 },
    { area: "Language", EE: 54, ME: 31, AE: 10, BE: 5 },
    { area: "Environmental", EE: 48, ME: 35, AE: 13, BE: 4 },
    { area: "Creative Arts", EE: 72, ME: 18, AE: 7, BE: 3 },
    { area: "CRE", EE: 44, ME: 38, AE: 14, BE: 4 },
    { area: "Movement", EE: 67, ME: 21, AE: 9, BE: 3 },
];

// ── 2B-2: Stream Performance ──
export const streamData: StreamRow[] = [
    { level: "EE", east: 38, west: 29 },
    { level: "ME", east: 41, west: 44 },
    { level: "AE", east: 15, west: 20 },
    { level: "BE", east: 6, west: 7 },
];

export const streams = ["Grade 4 East", "Grade 4 West"];

// ── 2B-3: YoY Comparison ──
export const yoyData: YoYRow[] = [
    { week: "Wk 1", thisYear: 3.1, lastYear: 2.8 },
    { week: "Wk 2", thisYear: 3.2, lastYear: 2.9 },
    { week: "Wk 3", thisYear: 3.4, lastYear: 3.0 },
    { week: "Wk 4", thisYear: 3.3, lastYear: 3.1 },
    { week: "Wk 5", thisYear: 3.5, lastYear: 3.1 },
    { week: "Wk 6", thisYear: 3.6, lastYear: 3.2 },
];

// ── 2B-4: Attendance vs CBC Score Scatter ──
export const scatterData: ScatterPoint[] = [
    { attendance: 99, score: 3.9 },
    { attendance: 95, score: 3.8 },
    { attendance: 91, score: 3.5 },
    { attendance: 88, score: 3.4 },
    { attendance: 84, score: 3.1 },
    { attendance: 80, score: 3.0 },
    { attendance: 75, score: 2.7 },
    { attendance: 72, score: 2.6 },
    { attendance: 68, score: 2.4 },
    { attendance: 61, score: 2.1 },
    { attendance: 55, score: 1.8 },
    { attendance: 48, score: 1.6 },
    { attendance: 43, score: 1.4 },
    { attendance: 40, score: 1.3 },
    // ~40 points total — programmatic generation for demo
    ...[...Array(26)].map(() => {
        const att = Math.round(40 + Math.random() * 55);
        return { attendance: att, score: Math.round((1 + ((att - 40) / 55) * 3) * 10) / 10 };
    }),
];

// ── 2C: System Compliance ──
export const complianceItems: ComplianceItem[] = [
    {
        id: "no-class",
        issue: "Students without a class",
        count: 6,
        severity: "warning",
        resolveLabel: "Assign classes",
    },
    {
        id: "no-student",
        issue: "Classes without students",
        count: 2,
        severity: "warning",
        resolveLabel: "Review classes",
    },
    {
        id: "no-teacher",
        issue: "Classes without a teacher",
        count: 1,
        severity: "error",
        resolveLabel: "Assign teacher",
    },
    {
        id: "no-class-t",
        issue: "Teachers without a class",
        count: 4,
        severity: "warning",
        resolveLabel: "Assign classes",
    },
    {
        id: "no-parent",
        issue: "Parents without a child assigned",
        count: 11,
        severity: "warning",
        resolveLabel: "Link guardians",
    },
];

// ── 2D-1: CBC Eval Compliance ──
export const evalCompliance: EvalCompliance[] = [
    { teacher: "Ms. Akinyi", class: "Grade 3 East", graded: 18, total: 24 },
    { teacher: "Mr. Kamau", class: "Grade 4 West", graded: 24, total: 30 },
    { teacher: "Ms. Wambui", class: "Grade 5 East", graded: 22, total: 22 },
    { teacher: "Mr. Odhiambo", class: "Grade 2 West", graded: 8, total: 28 },
];

// ── 2D-2: Weighting Audit ──
export const weightingIssues: WeightingIssue[] = [
    { class: "Grade 3 East", issue: "No END_TERM task configured", severity: "error" },
    { class: "Grade 2 West", issue: "CAT tasks exceed weight cap", severity: "warning" },
];

// ── 2D-3: Chronic Absentee Watchlist ──
export const absentees: AbsenteeRow[] = [
    { name: "James Mwangi", class: "Grade 4 East", absentDays: 14, threshold: 10 },
    { name: "Grace Atieno", class: "Grade 3 West", absentDays: 11, threshold: 10 },
    { name: "Peter Ndiwa", class: "Grade 5 East", absentDays: 10, threshold: 10 },
];

// ── 2E: Institutional Finance ──
export const finance: FinanceData = {
    outstandingTotal: 487350,
    invoiceCount: 94,
    totalInvoiced: 2150000,
    totalCollected: 1662650,
    currency: "KES",
    feeCategories: [
        { name: "Tuition", configured: true },
        { name: "Transport", configured: true },
        { name: "Lunch", configured: true },
        { name: "Activity", configured: false },
    ],
};

// ── 2F-1: Pre-invited User Queue ──
export const pendingUsers: PendingUser[] = [
    {
        name: "Peter Ndung'u",
        role: "TEACHER",
        email: "p.ndungu@school.ac.ke",
        invitedAt: "2026-06-14",
    },
    {
        name: "Susan Chebet",
        role: "SUPPORT_STAFF",
        email: "s.chebet@school.ac.ke",
        invitedAt: "2026-06-12",
    },
];

// ── 2F-2: Recent Medical Incidents ──
export const medicalIncidents: MedicalIncident[] = [
    {
        student: "Kevin Ochieng",
        class: "Grade 4 East",
        time: "09:14",
        symptoms: "Headache, dizziness",
        action: "Sent to sick bay, parents notified",
    },
    {
        student: "Mercy Wanjiku",
        class: "Grade 3 West",
        time: "11:32",
        symptoms: "Nosebleed",
        action: "First aid applied, resolved",
    },
];

// ── Week info for term banner ──
export const termWeekInfo = { currentWeek: 6, totalWeeks: 14 };
