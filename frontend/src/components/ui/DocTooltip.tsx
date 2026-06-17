"use client";

import React from "react";
import Link from "next/link";
import { HelpCircle } from "lucide-react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

interface DocTooltipProps {
    summary: string;
    slug: string;
    anchorId?: string;
}

export function DocTooltip({ summary, slug, anchorId }: DocTooltipProps) {
    const fullPath = `/docs/${slug}${anchorId ? `#${anchorId}` : ""}`;

    return (
        <TooltipProvider delayDuration={150}>
            <Tooltip>
                <TooltipTrigger asChild>
                    <span className="text-muted-foreground hover:text-foreground ml-1 inline-flex cursor-help items-center align-middle transition-colors">
                        <HelpCircle className="h-4 w-4" />
                    </span>
                </TooltipTrigger>
                <TooltipContent className="side-top flex max-w-xs flex-col space-y-2 p-3 text-xs">
                    <p className="block leading-relaxed">{summary}</p>
                    <Link
                        href={fullPath}
                        className="text-primary block font-semibold hover:underline"
                    >
                        View Full Docs →
                    </Link>
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
    );
}
