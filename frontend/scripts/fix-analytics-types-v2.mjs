import fs from "fs";

const base = "src/features/analytics";

// 1. Fix double-cast for `as Record<string, unknown>`
for (const f of [
    "cohort-position/class-score-scatter.tsx",
    "cohort-position/class-vs-grade-bar.tsx",
    "cohort-position/rank-over-terms-line.tsx",
]) {
    const fp = `${base}/components/${f}`;
    let c = fs.readFileSync(fp, "utf-8");
    c = c.replace(
        /\(_item as Record<string, unknown>\)/g,
        "(_item as unknown as Record<string, unknown>)"
    );
    fs.writeFileSync(fp, c);
    console.log(`Fixed cast in ${f}`);
}

// 2. Fix distribution-curve.tsx formatter
let dc = fs.readFileSync(`${base}/components/cohort-position/distribution-curve.tsx`, "utf-8");
dc = dc.replace(
    /formatter=\{(value: number\) => value\.toFixed\(4\)\}/,
    'formatter={(val: unknown) => { const v = Number(val); return isNaN(v) ? "" : v.toFixed(4); }}'
);
fs.writeFileSync(`${base}/components/cohort-position/distribution-curve.tsx`, dc);
console.log("Fixed distribution-curve formatter");

// 3. Fix rank-over-terms-line.tsx `return value;` -> `return val;`
let rot = fs.readFileSync(`${base}/components/cohort-position/rank-over-terms-line.tsx`, "utf-8");
rot = rot.replace(/\n                  return value;/, "\n                  return val;");
fs.writeFileSync(`${base}/components/cohort-position/rank-over-terms-line.tsx`, rot);
console.log("Fixed rank-over-terms value -> val");

// 4. Fix projection-scatter-trend.tsx - remove both color AND theme from chartConfig
let pst = fs.readFileSync(
    `${base}/components/performance-projections/projection-scatter-trend.tsx`,
    "utf-8"
);
pst = pst.replace(
    `    color: "hsl(var(--chart-4))",
    theme: { light: "#f59e0b", dark: "#f59e0b" },`,
    `    color: "hsl(var(--chart-4))",`
);
fs.writeFileSync(`${base}/components/performance-projections/projection-scatter-trend.tsx`, pst);
console.log("Fixed projection-scatter-trend chartConfig");

// 5. Fix Dot spread in projection-scatter-trend.tsx
pst = fs.readFileSync(
    `${base}/components/performance-projections/projection-scatter-trend.tsx`,
    "utf-8"
);
pst = pst.replace(
    /<Dot\s*\n\s*key=\{payload\.termName\}\s*\n\s*r=\{6\}\s*\n\s*fill="var\(--color-projected\)"\s*\n\s*stroke="hsl\(var\(--background\)\)"\s*\n\s*strokeWidth=\{2\}\s*\n\s*\/>/g,
    '<Dot key={payload.termName} r={6} fill="var(--color-projected)" stroke="hsl(var(--background))" strokeWidth={2} />'
);
fs.writeFileSync(`${base}/components/performance-projections/projection-scatter-trend.tsx`, pst);
console.log("Fixed projection-scatter-trend Dot spread");

// 6. Fix level-donut-chart.tsx arithmetic
let ldc = fs.readFileSync(`${base}/components/student-term-overall/level-donut-chart.tsx`, "utf-8");
ldc = ldc.replace(
    /formatter=\{(value: number, name: string\) => \{[\s\S]*?\}\}/,
    `formatter={(val, name) => { const value = Number(val); const pct = total > 0 ? ((value / total) * 100).toFixed(0) : "0"; return isNaN(value) ? "" : value + " subjects (" + pct + "%)"; }}`
);
fs.writeFileSync(`${base}/components/student-term-overall/level-donut-chart.tsx`, ldc);
console.log("Fixed level-donut-chart formatter");

// 7. Fix waterfall-contribution.tsx formatter
let wc = fs.readFileSync(
    `${base}/components/student-term-overall/waterfall-contribution.tsx`,
    "utf-8"
);
wc = wc.replace(
    /formatter=\{(value: number, name: string\) => \{[\s\S]*?\}\}/,
    `formatter={(val, name) => { if (name === "base") return null; const v = Number(val); return isNaN(v) ? "" : v.toFixed(1) + "%"; }}`
);
fs.writeFileSync(`${base}/components/student-term-overall/waterfall-contribution.tsx`, wc);
console.log("Fixed waterfall-contribution formatter");

// 8. Fix subject-dot-plot.tsx ZAxis range
let sdp = fs.readFileSync(`${base}/components/student-term-subject/subject-dot-plot.tsx`, "utf-8");
sdp = sdp.replace(/range=\{\[100\]\}/, "range={[100, 100]}");
fs.writeFileSync(`${base}/components/student-term-subject/subject-dot-plot.tsx`, sdp);
console.log("Fixed subject-dot-plot ZAxis");

// 9. Fix subject-treemap.tsx - data type + content return null
let stm = fs.readFileSync(`${base}/components/student-term-subject/subject-treemap.tsx`, "utf-8");
// Add index signature to type
stm = stm.replace(
    "export interface SubjectTreemapEntry {",
    "export interface SubjectTreemapEntry { [key: string]: unknown;"
);
// Fix content return type - use a fragment instead of null
stm = stm.replace("if (depth !== 1) return null;", "if (depth !== 1) return <React.Fragment />;");
// Add import for React.Fragment
stm = stm.replace(`import {`, `import React from "react";\nimport {`);
fs.writeFileSync(`${base}/components/student-term-subject/subject-treemap.tsx`, stm);
console.log("Fixed subject-treemap types");

// 10. Fix strand-treemap.tsx - data type + content return null
let strm = fs.readFileSync(`${base}/components/subject-strand/strand-treemap.tsx`, "utf-8");
// Add index signature to types
strm = strm.replace(
    "export interface SubStrandNode {",
    "export interface SubStrandNode { [key: string]: unknown;"
);
strm = strm.replace(
    "export interface StrandNode {",
    "export interface StrandNode { [key: string]: unknown;"
);
// Fix content return null
strm = strm.replace("if (depth === 0) return null;", "if (depth === 0) return <React.Fragment />;");
strm = strm.replace("import {", 'import React from "react";\nimport {');
fs.writeFileSync(`${base}/components/subject-strand/strand-treemap.tsx`, strm);
console.log("Fixed strand-treemap types");

// 11. Fix analytics/index.ts - duplicate TermOverTermComparison
let idx = fs.readFileSync(`${base}/index.ts`, "utf-8");
// It's exported from both barrels. Remove from student-term-overall since TermOverTermComparison
// is a generic type, not a component. Let me check which one is the component.
// Actually, looking at the file, it's the TermOverTermComparison component exported twice.
// Remove the duplicate from the student-term-overall section.
idx = idx.replace(
    `  TermOverTermComparison,
  TermOverTermComparisonSkeleton,`,
    `  TermOverTermComparisonSkeleton,`
);
fs.writeFileSync(`${base}/index.ts`, idx);
console.log("Fixed analytics/index.ts duplicate");

console.log("\nAll fixes applied.");
