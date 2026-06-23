"use client";

import { Fragment, useMemo, useState } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";

import {
    Breadcrumb,
    BreadcrumbList,
    BreadcrumbItem,
    BreadcrumbLink,
    BreadcrumbPage,
    BreadcrumbSeparator,
    BreadcrumbEllipsis,
} from "@/components/ui/breadcrumb";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

import { buildBreadcrumbs, type BreadcrumbSegment } from "@/lib/utils/breadcrumbs";

// ─── Props ──────────────────────────────────────────────────────────────────

export interface SmartBreadcrumbProps {
    /**
     * Maximum number of breadcrumb items to show before collapsing.
     * Must be >= 3 (Dashboard + ellipsis + current page minimum).
     * @default 4
     */
    maxVisible?: number;
}

// ─── Constants ──────────────────────────────────────────────────────────────

const MIN_VISIBLE = 3;

// ─── Component ──────────────────────────────────────────────────────────────

export function SmartBreadcrumb({ maxVisible = 4 }: SmartBreadcrumbProps) {
    const pathname = usePathname();

    const segments = useMemo<BreadcrumbSegment[]>(() => buildBreadcrumbs(pathname), [pathname]);

    const [dropdownOpen, setDropdownOpen] = useState(false);

    // ── Determine visible / hidden slices ─────────────────────────────────

    const effectiveMax = Math.max(maxVisible, MIN_VISIBLE);
    const needsCollapse = segments.length > effectiveMax;

    let visible: BreadcrumbSegment[];
    let hidden: BreadcrumbSegment[];

    if (needsCollapse) {
        const first = segments[0];
        const last = segments[segments.length - 1];
        const middle = segments.slice(1, segments.length - 1);

        visible = [first, last];
        // Hidden middle segments — listed in reverse order (deepest first)
        hidden = middle.slice().reverse();
    } else {
        visible = segments;
        hidden = [];
    }

    // ── Render helpers ────────────────────────────────────────────────────

    /**
     * Renders a single BreadcrumbSegment as either a linked item or the
     * current (non-linked) page.
     */
    function renderItem(segment: BreadcrumbSegment) {
        return (
            <BreadcrumbItem key={segment.href}>
                {segment.isLast ? (
                    <BreadcrumbPage>{segment.label}</BreadcrumbPage>
                ) : (
                    <BreadcrumbLink asChild>
                        <Link href={segment.href}>{segment.label}</Link>
                    </BreadcrumbLink>
                )}
            </BreadcrumbItem>
        );
    }

    /**
     * Renders the ellipsis dropdown trigger for hidden middle segments.
     */
    function renderEllipsis() {
        return (
            <Fragment key="ellipsis">
                <BreadcrumbSeparator />
                <BreadcrumbItem>
                    <DropdownMenu open={dropdownOpen} onOpenChange={setDropdownOpen}>
                        <DropdownMenuTrigger asChild>
                            <span>
                                <BreadcrumbEllipsis />
                                <span className="sr-only">Show hidden breadcrumbs</span>
                            </span>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="start">
                            {hidden.map((segment) => (
                                <DropdownMenuItem key={segment.href} asChild>
                                    <Link href={segment.href}>{segment.label}</Link>
                                </DropdownMenuItem>
                            ))}
                        </DropdownMenuContent>
                    </DropdownMenu>
                </BreadcrumbItem>
            </Fragment>
        );
    }

    // ── Render ────────────────────────────────────────────────────────────

    return (
        <Breadcrumb>
            <BreadcrumbList>
                {/* First visible item (always Dashboard) */}
                {renderItem(visible[0])}

                {/* Ellipsis between first and last when collapsed */}
                {needsCollapse && renderEllipsis()}

                {/* Remaining visible items (after first), with separators */}
                {visible.slice(1).map((segment) => (
                    <Fragment key={segment.href}>
                        <BreadcrumbSeparator />
                        {renderItem(segment)}
                    </Fragment>
                ))}
            </BreadcrumbList>
        </Breadcrumb>
    );
}
