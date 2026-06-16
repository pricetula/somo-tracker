'use client';

import React from 'react';
import Link from 'next/link';
import { HelpCircle } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

interface DocTooltipProps {
  summary: string;
  slug: string;
  anchorId?: string;
}

export function DocTooltip({ summary, slug, anchorId }: DocTooltipProps) {
  const fullPath = `/docs/${slug}${anchorId ? `#${anchorId}` : ''}`;

  return (
    <TooltipProvider delayDuration={150}>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex items-center text-muted-foreground hover:text-foreground transition-colors ml-1 cursor-help align-middle">
            <HelpCircle className="h-4 w-4" />
          </span>
        </TooltipTrigger>
        <TooltipContent className="max-w-xs p-3 space-y-2 text-xs side-top flex flex-col">
          <p className="leading-relaxed block">{summary}</p>
          <Link href={fullPath} className="font-semibold text-primary hover:underline block">
            View Full Docs →
          </Link>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
