/**
 * Class normalization and token-matching utilities.
 *
 * Implements the explicit Token Matcher Scoring algorithm:
 * 1. Tokenize raw string into lowercase alphanumeric pieces
 * 2. For each BackendClass, compute score:
 *    - +2 if numeric token matches grade_level digits
 *    - +2 if any raw token has ≥0.8 similarity to stream_name
 *    - +1 if raw token exactly equals a letter/word in stream_name
 * 3. Auto-accept only if score ≥ 4 and ≥ 2 points clear of runner-up
 * 4. Score 0 = unmatched (no plausible match)
 */

import type { Class } from "@/lib/api/generated";
import type { UnresolvedClassEntry, ClassMatchCandidate } from "../types";

// ─── Token Similarity (Levenshtein-based) ────────────────────────────────

/**
 * Compute Levenshtein distance between two strings.
 */
function levenshteinDistance(a: string, b: string): number {
    const m = a.length;
    const n = b.length;
    const dp: number[][] = [];

    for (let i = 0; i <= m; i++) {
        dp[i] = [i];
    }
    for (let j = 0; j <= n; j++) {
        dp[0][j] = j;
    }

    for (let i = 1; i <= m; i++) {
        for (let j = 1; j <= n; j++) {
            const cost = a[i - 1] === b[j - 1] ? 0 : 1;
            dp[i][j] = Math.min(dp[i - 1][j] + 1, dp[i][j - 1] + 1, dp[i - 1][j - 1] + cost);
        }
    }

    return dp[m][n];
}

/**
 * Compute string similarity between 0 and 1 using Levenshtein distance.
 * 1.0 = exact match, 0.0 = completely different.
 */
function stringSimilarity(a: string, b: string): number {
    if (a === b) return 1.0;
    if (a.length === 0 || b.length === 0) return 0.0;
    const dist = levenshteinDistance(a, b);
    const maxLen = Math.max(a.length, b.length);
    return 1 - dist / maxLen;
}

// ─── Tokenization ─────────────────────────────────────────────────────────

/**
 * Tokenize a raw class string into lowercase alphanumeric pieces.
 * e.g. "2 A Simba" → ["2", "a", "simba"]
 * e.g. "Simba 2A"  → ["simba", "2a"]
 */
function tokenize(raw: string): string[] {
    return raw
        .toLowerCase()
        .split(/[\s\-_/]+/)
        .filter((t) => t.length > 0);
}

// ─── Scoring ──────────────────────────────────────────────────────────────

interface ScoredCandidate {
    classId: string;
    displayLabel: string;
    score: number;
}

/**
 * Score a single raw token against a backend class.
 */
function scoreTokenAgainstClass(token: string, classInfo: Class): number {
    let score = 0;

    // Check if token is numeric
    const isNumeric = /^\d+$/.test(token);

    // +2 if numeric token matches grade_level digits
    if (isNumeric) {
        const gradeDigits = classInfo.grade_level.replace(/\D/g, "");
        if (gradeDigits.length > 0 && token === gradeDigits) {
            score += 2;
        }
    }

    // +2 if token has ≥0.8 similarity to stream_name
    const streamSimilarity = stringSimilarity(token, classInfo.stream_name.toLowerCase());
    if (streamSimilarity >= 0.8) {
        score += 2;
    }

    // +1 if token exactly equals a letter/word in stream_name
    const streamTokens = classInfo.stream_name.toLowerCase().split(/[\s\-_/]+/);
    if (streamTokens.some((st) => st === token)) {
        score += 1;
    }

    // Also check if token matches display_label substrings
    const displayTokens = classInfo.display_label.toLowerCase().split(/[\s\-_/]+/);
    if (displayTokens.some((dt) => dt === token)) {
        score += 1;
    }

    return score;
}

/**
 * Compute total score for a raw string against a backend class.
 */
function scoreRawStringAgainstClass(raw: string, classInfo: Class): number {
    const tokens = tokenize(raw);
    let totalScore = 0;

    for (const token of tokens) {
        totalScore += scoreTokenAgainstClass(token, classInfo);
    }

    return totalScore;
}

// ─── Main Resolution Function ─────────────────────────────────────────────

/**
 * Resolve all unique raw class strings against the available backend classes.
 *
 * @param rawStrings - Set of unique raw class strings from the spreadsheet
 * @param classes - Array of backend Class objects
 * @returns Array of UnresolvedClassEntry with candidates and auto-resolution
 */
export function resolveClassStrings(
    rawStrings: Set<string>,
    classes: Class[]
): UnresolvedClassEntry[] {
    const entries: UnresolvedClassEntry[] = [];

    for (const raw of rawStrings) {
        if (!raw.trim()) {
            entries.push({
                raw_string: raw,
                count: 0,
                status: "unmatched",
                candidates: [],
                resolved_id: null,
            });
            continue;
        }

        // Score against every class
        const scored: ScoredCandidate[] = classes.map((c) => ({
            classId: c.id,
            displayLabel: c.display_label,
            score: scoreRawStringAgainstClass(raw, c),
        }));

        // Sort descending by score
        scored.sort((a, b) => b.score - a.score);

        const candidates: ClassMatchCandidate[] = scored
            .filter((s) => s.score > 0)
            .map((s) => ({
                class_id: s.classId,
                display_label: s.displayLabel,
                score: s.score,
            }));

        // Determine status
        let status: "matched" | "ambiguous" | "unmatched";
        let resolvedId: string | null = null;

        if (candidates.length === 0) {
            status = "unmatched";
        } else if (scored[0].score >= 4) {
            // Check if top score is ≥2 points clear of second
            const topScore = scored[0].score;
            const secondScore = scored.length > 1 ? scored[1].score : -1;
            if (topScore - secondScore >= 2) {
                status = "matched";
                resolvedId = scored[0].classId;
            } else {
                status = "ambiguous";
            }
        } else {
            status = "ambiguous";
        }

        entries.push({
            raw_string: raw,
            count: 0, // caller sets count based on actual row occurrences
            status,
            candidates,
            resolved_id: resolvedId,
        });
    }

    return entries;
}

/**
 * Compute occurrence counts for each raw class string from the spreadsheet rows.
 */
export function countClassOccurrences(
    rows: Record<string, string>[],
    classColumn: string
): Map<string, number> {
    const counts = new Map<string, number>();
    for (const row of rows) {
        const value = (row[classColumn] ?? "").trim();
        if (value.length > 0) {
            counts.set(value, (counts.get(value) ?? 0) + 1);
        }
    }
    return counts;
}

/**
 * Merge occurrence counts into unresolved class entries.
 */
export function mergeCountsIntoEntries(
    entries: UnresolvedClassEntry[],
    counts: Map<string, number>
): UnresolvedClassEntry[] {
    return entries.map((entry) => ({
        ...entry,
        count: counts.get(entry.raw_string) ?? 0,
    }));
}

/**
 * Get the reason text for a class entry status.
 */
export function getClassStatusReason(entry: UnresolvedClassEntry): string {
    switch (entry.status) {
        case "matched":
            return "Auto-matched";
        case "unmatched":
            return "No matching class found";
        case "ambiguous":
            if (entry.candidates.length >= 2) {
                const topTwo = entry.candidates.slice(0, 2);
                return `Ambiguous between: ${topTwo.map((c) => c.display_label).join(" and ")}`;
            }
            return "Low confidence match — please review";
    }
}
