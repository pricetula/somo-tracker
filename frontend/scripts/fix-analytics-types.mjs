import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const base = path.resolve(__dirname, "../src/features/analytics/components");

const files = [
    "attendance-term/attendance-term-trend-line.tsx",
    "attendance-term/attendance-subject-comparison-bar.tsx",
    "attendance-term/attendance-vs-overall-scatter.tsx",
    "class-daily-attendance/daily-line-chart.tsx",
    "class-daily-attendance/day-of-week-bar.tsx",
    "student-term-subject/subject-radar-chart.tsx",
    "student-term-subject/subject-comparison-bar.tsx",
    "student-term-subject/subject-dot-plot.tsx",
    "student-term-subject/source-composition-stacked-bar.tsx",
    "student-term-overall/term-over-term-comparison.tsx",
    "student-term-overall/level-distribution-bar.tsx",
    "student-term-overall/level-donut-chart.tsx",
    "student-term-overall/waterfall-contribution.tsx",
    "cohort-position/rank-over-terms-line.tsx",
    "cohort-position/class-vs-grade-bar.tsx",
    "cohort-position/class-score-scatter.tsx",
    "cohort-position/distribution-curve.tsx",
    "subject-strand/strand-mastery-bar.tsx",
    "subject-strand/skill-radar.tsx",
    "subject-strand/level-pie-chart.tsx",
    "subject-strand/before-after-comparison.tsx",
    "performance-projections/projection-scatter-trend.tsx",
    "performance-projections/actual-to-projected-waterfall.tsx",
    "performance-projections/projection-table-sparklines.tsx",
];

let fixedCount = 0;

// Pattern 1: formatter={(value: number) => `...`}
// -> formatter={(value) => { if (value == null || typeof value !== "number") return ""; return `...`; }}
const pattern1 = /\(value: number\)\s*=>\s*`([^`]*)`/g;

// Pattern 2: formatter={(value: number, name: string) => { ... }}
// -> formatter={(value, name) => { if (value == null) return ""; ... }}
const pattern2 = /\(value: number,\s*name: string\)\s*=>\s*\{/g;

for (const rel of files) {
    const fp = path.join(base, rel);
    if (!fs.existsSync(fp)) continue;

    let content = fs.readFileSync(fp, "utf-8");
    const original = content;

    content = content.replace(pattern1, (match, template) => {
        const escaped = template.replace(/`/g, "\\`");
        return `(value) => { if (value == null || typeof value !== "number") return ""; return \`${escaped}\`; }`;
    });

    content = content.replace(pattern2, () => {
        return '(value, name) => { if (value == null) return "";';
    });

    if (content !== original) {
        fs.writeFileSync(fp, content);
        console.log("Fixed: " + rel);
        fixedCount++;
    }
}

// Specific fixes
// Dot component spread issue
const trendFile = path.join(base, "attendance-term/attendance-term-trend-line.tsx");
let tl = fs.readFileSync(trendFile, "utf-8");
const tlOrig = tl;
tl = tl.replace(/\.\.\.props\}\)\)/, "})");
if (tl !== tlOrig) {
    fs.writeFileSync(trendFile, tl);
    console.log("Fixed Dot spread in trend-line");
}

// ZAxis range needs [min, max] not just [number]
const scatterFile = path.join(base, "attendance-term/attendance-vs-overall-scatter.tsx");
let sc = fs.readFileSync(scatterFile, "utf-8");
const scOrig = sc;
sc = sc.replace(/range=\{\[(\d+)\]\}/g, "range={[$1, $1]}");
if (sc !== scOrig) {
    fs.writeFileSync(scatterFile, sc);
    console.log("Fixed ZAxis range in scatter");
}

console.log(`\nDone. Fixed ${fixedCount} files.`);
