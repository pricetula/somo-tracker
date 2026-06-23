"use client";

import { usePathname } from "next/navigation";
import Link from "next/link";
import { useEffect, useRef, useState, Fragment } from "react";
import {
    Breadcrumb,
    BreadcrumbList,
    BreadcrumbItem,
    BreadcrumbLink,
    BreadcrumbSeparator,
    BreadcrumbPage,
    BreadcrumbEllipsis,
} from "@/components/ui/breadcrumb";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

// ─── Types ──────────────────────────────────────────────────────────────────

interface BreadcrumbSegment {
    label: string; // display text
    href: string; // full path up to this segment e.g. "/students/uuid-123"
    isUUID: boolean; // true if this segment is a UUID
}

export interface SmartBreadcrumbProps {
    /** Optional path segments to use instead of the current URL pathname. */
    customSegments?: string[];
    className?: string;
}

// ─── Constants ──────────────────────────────────────────────────────────────

const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const UUID_REGEX_NODASH = /^[0-9a-f]{32}$/i;

const ROUTE_LABELS: Record<string, string> = {
    students: "Students",
    assessments: "Assessments",
    questions: "Questions",
    courses: "Courses",
    modules: "Modules",
    submissions: "Submissions",
    users: "Users",
    assignments: "Assignments",
};

const SINGULAR_LABELS: Record<string, string> = {
    students: "Student",
    assessments: "Assessment",
    questions: "Question",
    courses: "Course",
    modules: "Module",
    submissions: "Submission",
    users: "User",
    assignments: "Assignment",
};

// ─── Helpers ─────────────────────────────────────────────────────────────────

function isUUID(segment: string): boolean {
    return UUID_REGEX.test(segment) || UUID_REGEX_NODASH.test(segment);
}

function singularize(word: string): string {
    const lower = word.toLowerCase();
    if (SINGULAR_LABELS[lower]) return SINGULAR_LABELS[lower];
    if (lower.endsWith("ies")) return capitalize(lower.slice(0, -3) + "y");
    if (lower.endsWith("es")) return capitalize(lower.slice(0, -2));
    if (lower.endsWith("s")) return capitalize(lower.slice(0, -1));
    return capitalize(lower);
}

function capitalize(str: string): string {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Returns a human-readable label for a path segment.
 * UUID segments are labelled by singularizing the preceding segment (their
 * collection name). If no parent segment exists, falls back to "Item".
 */
function labelSegment(segment: string, prevSegment: string | undefined): string {
    if (isUUID(segment)) {
        // FIX: use "Item" instead of the vague "Detail" as fallback
        return prevSegment ? singularize(prevSegment) : "Item";
    }
    const lower = segment.toLowerCase();
    return ROUTE_LABELS[lower] ?? capitalize(segment);
}

// ─── Parse path into segments ─────────────────────────────────────────────

/**
 * Parses a pathname string (e.g. "/students/uuid/assessments") into an array
 * of BreadcrumbSegments, always starting with a "Dashboard" root entry.
 */
function parsePath(pathname: string): BreadcrumbSegment[] {
    const parts = pathname.split("/").filter(Boolean);

    const segments: BreadcrumbSegment[] = [{ label: "Dashboard", href: "/", isUUID: false }];

    let cumulativePath = "";

    parts.forEach((part, index) => {
        cumulativePath += "/" + part;
        const prevPart = parts[index - 1];
        segments.push({
            label: labelSegment(part, prevPart),
            href: cumulativePath,
            isUUID: isUUID(part),
        });
    });

    return segments;
}

// ─── Component ────────────────────────────────────────────────────────────

export function SmartBreadcrumb({ customSegments, className }: SmartBreadcrumbProps) {
    const rawPathname = usePathname();

    // customSegments may contain empty strings (e.g. from a split); filter them out.
    const effectivePath = customSegments
        ? "/" + customSegments.filter(Boolean).join("/")
        : (rawPathname ?? "/");

    const allSegments = parsePath(effectivePath);

    const navRef = useRef<HTMLElement>(null);

    // FIX: track the number of *middle* items to show, not total visible count.
    // This avoids the off-by-one confusion between first/last and middle items.
    const middleAll = allSegments.slice(1, allSegments.length > 1 ? -1 : undefined);
    const [middleShownCount, setMiddleShownCount] = useState(middleAll.length);

    // Attach a ResizeObserver that adjusts middleShownCount one step at a time.
    // No useCallback needed — the function is defined directly inside the effect
    // so the React Compiler can analyse and optimise it freely.
    useEffect(() => {
        const nav = navRef.current;
        if (!nav) return;

        let rafId: number;

        const handleResize = () => {
            cancelAnimationFrame(rafId);
            rafId = requestAnimationFrame(() => {
                const el = navRef.current;
                if (!el) return;

                if (el.scrollWidth <= el.clientWidth) {
                    // Fits — try to restore one hidden middle item.
                    setMiddleShownCount((prev) => {
                        const max = Math.max(0, allSegments.length - 2);
                        return Math.min(prev + 1, max);
                    });
                } else {
                    // Overflows — hide one more middle item.
                    setMiddleShownCount((prev) => Math.max(0, prev - 1));
                }
            });
        };

        const observer = new ResizeObserver(handleResize);
        observer.observe(nav);
        handleResize(); // measure immediately on mount

        return () => {
            observer.disconnect();
            cancelAnimationFrame(rafId);
        };
         
    }, [allSegments.length]); // re-attach when segment count changes (route change)

    // ── Derived display slices ───────────────────────────────────────────

    const isSingleItem = allSegments.length === 1;
    const first = allSegments[0];
    // FIX: guard last so it is only set when there is more than one segment.
    const last = isSingleItem ? null : allSegments[allSegments.length - 1];

    const clampedShownCount = Math.max(0, Math.min(middleShownCount, middleAll.length));
    const middleVisible = middleAll.slice(0, clampedShownCount);
    const middleHidden = middleAll.slice(clampedShownCount);
    const hasEllipsis = middleHidden.length > 0;

    return (
        // FIX: use <nav> with overflow-hidden; separators & items no longer
        // wrapped in <span> (replaced with Fragment) to keep flex semantics correct.
        <nav
            ref={navRef}
            className={`w-full overflow-hidden ${className ?? ""}`}
            aria-label="Breadcrumb"
        >
            <Breadcrumb>
                <BreadcrumbList className="flex-nowrap overflow-hidden">
                    {/* FIRST ITEM — always Dashboard */}
                    <BreadcrumbItem>
                        {isSingleItem ? (
                            // FIX: add aria-current="page" when Dashboard is the current page
                            <BreadcrumbPage aria-current="page">{first.label}</BreadcrumbPage>
                        ) : (
                            <BreadcrumbLink asChild>
                                <Link href={first.href}>{first.label}</Link>
                            </BreadcrumbLink>
                        )}
                    </BreadcrumbItem>

                    {/* VISIBLE MIDDLE ITEMS */}
                    {/* FIX: use Fragment instead of <span> so flex layout is not broken */}
                    {middleVisible.map((seg) => (
                        <Fragment key={seg.href}>
                            <BreadcrumbSeparator />
                            <BreadcrumbItem>
                                <BreadcrumbLink asChild>
                                    <Link href={seg.href}>{seg.label}</Link>
                                </BreadcrumbLink>
                            </BreadcrumbItem>
                        </Fragment>
                    ))}

                    {/* ELLIPSIS DROPDOWN for hidden middle items */}
                    {hasEllipsis && (
                        <Fragment>
                            <BreadcrumbSeparator />
                            <BreadcrumbItem>
                                <DropdownMenu>
                                    <DropdownMenuTrigger asChild>
                                        <BreadcrumbEllipsis
                                            className="cursor-pointer"
                                            // FIX: meaningful aria-label for screen readers
                                            aria-label="Show more breadcrumb items"
                                        />
                                    </DropdownMenuTrigger>
                                    <DropdownMenuContent align="start">
                                        {middleHidden.map((seg) => (
                                            <DropdownMenuItem key={seg.href} asChild>
                                                <Link href={seg.href}>{seg.label}</Link>
                                            </DropdownMenuItem>
                                        ))}
                                    </DropdownMenuContent>
                                </DropdownMenu>
                            </BreadcrumbItem>
                        </Fragment>
                    )}

                    {/* LAST ITEM — current page, always visible, non-clickable */}
                    {last && (
                        <Fragment>
                            <BreadcrumbSeparator />
                            <BreadcrumbItem>
                                {/* FIX: aria-current="page" marks the active page for a11y */}
                                <BreadcrumbPage aria-current="page">{last.label}</BreadcrumbPage>
                            </BreadcrumbItem>
                        </Fragment>
                    )}
                </BreadcrumbList>
            </Breadcrumb>
        </nav>
    );
}
